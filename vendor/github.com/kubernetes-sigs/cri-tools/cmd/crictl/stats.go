/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package crictl

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"
	"time"

	units "github.com/docker/go-units"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"golang.org/x/net/context"
	pb "k8s.io/kubernetes/pkg/kubelet/apis/cri/runtime/v1alpha2"
)

type statsOptions struct {
	// all containers
	all bool
	// id of container
	id string
	// podID of container
	podID string
	// sample is the duration for sampling cpu usage.
	sample time.Duration
	// labels are selectors for the sandbox
	labels map[string]string
	// output format
	output string
}

var statsCommand = cli.Command{
	Name: "stats",
	// TODO(random-liu): Support live monitoring of resource usage.
	Usage:                  "List container(s) resource usage statistics",
	SkipArgReorder:         true,
	UseShortOptionHandling: true,
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "all, a",
			Usage: "Show all containers (default shows just running)",
		},
		cli.StringFlag{
			Name:  "id",
			Value: "",
			Usage: "Filter by container id",
		},
		cli.StringFlag{
			Name:  "pod, p",
			Value: "",
			Usage: "Filter by pod id",
		},
		cli.StringSliceFlag{
			Name:  "label",
			Usage: "Filter by key=value label",
		},
		cli.StringFlag{
			Name:  "output, o",
			Usage: "Output format, One of: json|yaml|table",
		},
		cli.IntFlag{
			Name:  "seconds, s",
			Value: 1,
			Usage: "Sample duration for CPU usage in seconds",
		},
	},
	Action: func(context *cli.Context) error {
		var err error
		if err = getRuntimeClient(context); err != nil {
			return err
		}

		opts := statsOptions{
			all:    context.Bool("all"),
			id:     context.String("id"),
			podID:  context.String("pod"),
			sample: time.Duration(context.Int("seconds")) * time.Second,
			output: context.String("output"),
		}
		opts.labels, err = parseLabelStringSlice(context.StringSlice("label"))
		if err != nil {
			return err
		}

		if err = ContainerStats(runtimeClient, opts); err != nil {
			return fmt.Errorf("get container stats failed: %v", err)
		}
		return nil
	},
}

type containerStatsByID []*pb.ContainerStats

func (c containerStatsByID) Len() int      { return len(c) }
func (c containerStatsByID) Swap(i, j int) { c[i], c[j] = c[j], c[i] }
func (c containerStatsByID) Less(i, j int) bool {
	return c[i].Attributes.Id < c[j].Attributes.Id
}

// ContainerStats sends a ListContainerStatsRequest to the server, and
// parses the returned ListContainerStatsResponse.
func ContainerStats(client pb.RuntimeServiceClient, opts statsOptions) error {
	filter := &pb.ContainerStatsFilter{}
	if opts.id != "" {
		filter.Id = opts.id
	}
	if opts.podID != "" {
		filter.PodSandboxId = opts.podID
	}
	if opts.labels != nil {
		filter.LabelSelector = opts.labels
	}
	request := &pb.ListContainerStatsRequest{
		Filter: filter,
	}
	logrus.Debugf("ListContainerStatsRequest: %v", request)
	r, err := client.ListContainerStats(context.Background(), request)
	logrus.Debugf("ListContainerResponse: %v", r)
	if err != nil {
		return err
	}
	sort.Sort(containerStatsByID(r.Stats))

	switch opts.output {
	case "json":
		return outputProtobufObjAsJSON(r)
	case "yaml":
		return outputProtobufObjAsYAML(r)
	}
	oldStats := make(map[string]*pb.ContainerStats)
	for _, s := range r.GetStats() {
		oldStats[s.Attributes.Id] = s
	}

	time.Sleep(opts.sample)

	logrus.Debugf("ListContainerStatsRequest: %v", request)
	r, err = client.ListContainerStats(context.Background(), request)
	logrus.Debugf("ListContainerResponse: %v", r)
	if err != nil {
		return err
	}
	sort.Sort(containerStatsByID(r.Stats))

	w := tabwriter.NewWriter(os.Stdout, 20, 1, 3, ' ', 0)
	// Use `+` to work around go vet bug
	fmt.Fprintln(w, "CONTAINER\tCPU %"+"\tMEM\tDISK\tINODES")
	for _, s := range r.GetStats() {
		id := getTruncatedID(s.Attributes.Id, "")
		cpu := s.GetCpu().GetUsageCoreNanoSeconds().GetValue()
		mem := s.GetMemory().GetWorkingSetBytes().GetValue()
		disk := s.GetWritableLayer().GetUsedBytes().GetValue()
		inodes := s.GetWritableLayer().GetInodesUsed().GetValue()
		if !opts.all && cpu == 0 && mem == 0 {
			// Skip non-running container
			continue
		}
		old, ok := oldStats[s.Attributes.Id]
		if !ok {
			// Skip new container
			continue
		}
		var cpuPerc float64
		if cpu != 0 {
			// Only generate cpuPerc for running container
			duration := s.GetCpu().GetTimestamp() - old.GetCpu().GetTimestamp()
			if duration == 0 {
				return fmt.Errorf("cpu stat is not updated during sample")
			}
			cpuPerc = float64(cpu-old.GetCpu().GetUsageCoreNanoSeconds().GetValue()) / float64(duration) * 100
		}
		fmt.Fprintf(w, "%s\t%.2f\t%s\t%s\t%d\n", id, cpuPerc, units.HumanSize(float64(mem)), units.HumanSize(float64(disk)), inodes)
	}

	w.Flush()
	return nil
}
