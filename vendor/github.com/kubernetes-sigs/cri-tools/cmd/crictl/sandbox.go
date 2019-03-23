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
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	units "github.com/docker/go-units"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"golang.org/x/net/context"

	pb "k8s.io/kubernetes/pkg/kubelet/apis/cri/runtime/v1alpha2"
)

type sandboxByCreated []*pb.PodSandbox

func (a sandboxByCreated) Len() int      { return len(a) }
func (a sandboxByCreated) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a sandboxByCreated) Less(i, j int) bool {
	return a[i].CreatedAt > a[j].CreatedAt
}

var runPodCommand = cli.Command{
	Name:      "runp",
	Usage:     "Run a new pod",
	ArgsUsage: "pod-config.[json|yaml]",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "runtime, r",
			Usage: "Runtime handler to use. Available options are defined by the container runtime.",
		},
	},

	Action: func(context *cli.Context) error {
		sandboxSpec := context.Args().First()
		if sandboxSpec == "" {
			return cli.ShowSubcommandHelp(context)
		}

		if err := getRuntimeClient(context); err != nil {
			return err
		}

		podSandboxConfig, err := loadPodSandboxConfig(sandboxSpec)
		if err != nil {
			return fmt.Errorf("load podSandboxConfig failed: %v", err)
		}

		// Test RuntimeServiceClient.RunPodSandbox
		err = RunPodSandbox(runtimeClient, podSandboxConfig, context.String("runtime"))
		if err != nil {
			return fmt.Errorf("run pod sandbox failed: %v", err)
		}
		return nil
	},
}

var stopPodCommand = cli.Command{
	Name:      "stopp",
	Usage:     "Stop one or more running pods",
	ArgsUsage: "POD-ID [POD-ID...]",
	Action: func(context *cli.Context) error {
		if context.NArg() == 0 {
			return cli.ShowSubcommandHelp(context)
		}
		if err := getRuntimeClient(context); err != nil {
			return err
		}
		for i := 0; i < context.NArg(); i++ {
			id := context.Args().Get(i)
			err := StopPodSandbox(runtimeClient, id)
			if err != nil {
				return fmt.Errorf("stopping the pod sandbox %q failed: %v", id, err)
			}
		}
		return nil
	},
}

var removePodCommand = cli.Command{
	Name:      "rmp",
	Usage:     "Remove one or more pods",
	ArgsUsage: "POD-ID [POD-ID...]",
	Action: func(context *cli.Context) error {
		if context.NArg() == 0 {
			return cli.ShowSubcommandHelp(context)
		}
		if err := getRuntimeClient(context); err != nil {
			return err
		}
		for i := 0; i < context.NArg(); i++ {
			id := context.Args().Get(i)
			err := RemovePodSandbox(runtimeClient, id)
			if err != nil {
				return fmt.Errorf("removing the pod sandbox %q failed: %v", id, err)
			}
		}
		return nil
	},
}

var podStatusCommand = cli.Command{
	Name:                   "inspectp",
	Usage:                  "Display the status of one or more pods",
	ArgsUsage:              "POD-ID [POD-ID...]",
	SkipArgReorder:         true,
	UseShortOptionHandling: true,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "output, o",
			Usage: "Output format, One of: json|yaml|table",
		},
		cli.BoolFlag{
			Name:  "quiet, q",
			Usage: "Do not show verbose information",
		},
	},
	Action: func(context *cli.Context) error {
		if context.NArg() == 0 {
			return cli.ShowSubcommandHelp(context)
		}
		if err := getRuntimeClient(context); err != nil {
			return err
		}
		for i := 0; i < context.NArg(); i++ {
			id := context.Args().Get(i)

			err := PodSandboxStatus(runtimeClient, id, context.String("output"), context.Bool("quiet"))
			if err != nil {
				return fmt.Errorf("getting the pod sandbox status for %q failed: %v", id, err)
			}
		}
		return nil
	},
}

var listPodCommand = cli.Command{
	Name:                   "pods",
	Usage:                  "List pods",
	SkipArgReorder:         true,
	UseShortOptionHandling: true,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "id",
			Value: "",
			Usage: "filter by pod id",
		},
		cli.StringFlag{
			Name:  "name",
			Value: "",
			Usage: "filter by pod name regular expression pattern",
		},
		cli.StringFlag{
			Name:  "namespace",
			Value: "",
			Usage: "filter by pod namespace regular expression pattern",
		},
		cli.StringFlag{
			Name:  "state, s",
			Value: "",
			Usage: "filter by pod state",
		},
		cli.StringSliceFlag{
			Name:  "label",
			Usage: "filter by key=value label",
		},
		cli.BoolFlag{
			Name:  "verbose, v",
			Usage: "show verbose info for pods",
		},
		cli.BoolFlag{
			Name:  "quiet, q",
			Usage: "list only pod IDs",
		},
		cli.StringFlag{
			Name:  "output, o",
			Usage: "Output format, One of: json|yaml|table",
		},
		cli.BoolFlag{
			Name:  "latest, l",
			Usage: "Show the most recently created pod",
		},
		cli.IntFlag{
			Name:  "last, n",
			Usage: "Show last n recently created pods. Set 0 for unlimited",
		},
		cli.BoolFlag{
			Name:  "no-trunc",
			Usage: "Show output without truncating the ID",
		},
	},
	Action: func(context *cli.Context) error {
		var err error
		if err = getRuntimeClient(context); err != nil {
			return err
		}

		opts := listOptions{
			id:                 context.String("id"),
			state:              context.String("state"),
			verbose:            context.Bool("verbose"),
			quiet:              context.Bool("quiet"),
			output:             context.String("output"),
			latest:             context.Bool("latest"),
			last:               context.Int("last"),
			noTrunc:            context.Bool("no-trunc"),
			nameRegexp:         context.String("name"),
			podNamespaceRegexp: context.String("namespace"),
		}
		opts.labels, err = parseLabelStringSlice(context.StringSlice("label"))
		if err != nil {
			return err
		}
		if err = ListPodSandboxes(runtimeClient, opts); err != nil {
			return fmt.Errorf("listing pod sandboxes failed: %v", err)
		}
		return nil
	},
}

// RunPodSandbox sends a RunPodSandboxRequest to the server, and parses
// the returned RunPodSandboxResponse.
func RunPodSandbox(client pb.RuntimeServiceClient, config *pb.PodSandboxConfig, runtime string) error {
	request := &pb.RunPodSandboxRequest{
		Config:         config,
		RuntimeHandler: runtime,
	}
	logrus.Debugf("RunPodSandboxRequest: %v", request)
	r, err := client.RunPodSandbox(context.Background(), request)
	logrus.Debugf("RunPodSandboxResponse: %v", r)
	if err != nil {
		return err
	}
	fmt.Println(r.PodSandboxId)
	return nil
}

// StopPodSandbox sends a StopPodSandboxRequest to the server, and parses
// the returned StopPodSandboxResponse.
func StopPodSandbox(client pb.RuntimeServiceClient, ID string) error {
	if ID == "" {
		return fmt.Errorf("ID cannot be empty")
	}
	request := &pb.StopPodSandboxRequest{PodSandboxId: ID}
	logrus.Debugf("StopPodSandboxRequest: %v", request)
	r, err := client.StopPodSandbox(context.Background(), request)
	logrus.Debugf("StopPodSandboxResponse: %v", r)
	if err != nil {
		return err
	}

	fmt.Printf("Stopped sandbox %s\n", ID)
	return nil
}

// RemovePodSandbox sends a RemovePodSandboxRequest to the server, and parses
// the returned RemovePodSandboxResponse.
func RemovePodSandbox(client pb.RuntimeServiceClient, ID string) error {
	if ID == "" {
		return fmt.Errorf("ID cannot be empty")
	}
	request := &pb.RemovePodSandboxRequest{PodSandboxId: ID}
	logrus.Debugf("RemovePodSandboxRequest: %v", request)
	r, err := client.RemovePodSandbox(context.Background(), request)
	logrus.Debugf("RemovePodSandboxResponse: %v", r)
	if err != nil {
		return err
	}
	fmt.Printf("Removed sandbox %s\n", ID)
	return nil
}

// marshalPodSandboxStatus converts pod sandbox status into string and converts
// the timestamps into readable format.
func marshalPodSandboxStatus(ps *pb.PodSandboxStatus) (string, error) {
	statusStr, err := protobufObjectToJSON(ps)
	if err != nil {
		return "", err
	}
	jsonMap := make(map[string]interface{})
	err = json.Unmarshal([]byte(statusStr), &jsonMap)
	if err != nil {
		return "", err
	}
	jsonMap["createdAt"] = time.Unix(0, ps.CreatedAt).Format(time.RFC3339Nano)
	return marshalMapInOrder(jsonMap, *ps)
}

// PodSandboxStatus sends a PodSandboxStatusRequest to the server, and parses
// the returned PodSandboxStatusResponse.
func PodSandboxStatus(client pb.RuntimeServiceClient, ID, output string, quiet bool) error {
	verbose := !(quiet)
	if output == "" { // default to json output
		output = "json"
	}
	if ID == "" {
		return fmt.Errorf("ID cannot be empty")
	}

	request := &pb.PodSandboxStatusRequest{
		PodSandboxId: ID,
		Verbose:      verbose,
	}
	logrus.Debugf("PodSandboxStatusRequest: %v", request)
	r, err := client.PodSandboxStatus(context.Background(), request)
	logrus.Debugf("PodSandboxStatusResponse: %v", r)
	if err != nil {
		return err
	}

	status, err := marshalPodSandboxStatus(r.Status)
	if err != nil {
		return err
	}
	switch output {
	case "json", "yaml":
		return outputStatusInfo(status, r.Info, output)
	case "table": // table output is after this switch block
	default:
		return fmt.Errorf("output option cannot be %s", output)
	}

	// output in table format by default.
	fmt.Printf("ID: %s\n", r.Status.Id)
	if r.Status.Metadata != nil {
		if r.Status.Metadata.Name != "" {
			fmt.Printf("Name: %s\n", r.Status.Metadata.Name)
		}
		if r.Status.Metadata.Uid != "" {
			fmt.Printf("UID: %s\n", r.Status.Metadata.Uid)
		}
		if r.Status.Metadata.Namespace != "" {
			fmt.Printf("Namespace: %s\n", r.Status.Metadata.Namespace)
		}
		fmt.Printf("Attempt: %v\n", r.Status.Metadata.Attempt)
	}
	fmt.Printf("Status: %s\n", r.Status.State)
	ctm := time.Unix(0, r.Status.CreatedAt)
	fmt.Printf("Created: %v\n", ctm)

	if r.Status.Network != nil {
		fmt.Printf("IP Address: %v\n", r.Status.Network.Ip)
	}
	if r.Status.Labels != nil {
		fmt.Println("Labels:")
		for _, k := range getSortedKeys(r.Status.Labels) {
			fmt.Printf("\t%s -> %s\n", k, r.Status.Labels[k])
		}
	}
	if r.Status.Annotations != nil {
		fmt.Println("Annotations:")
		for _, k := range getSortedKeys(r.Status.Annotations) {
			fmt.Printf("\t%s -> %s\n", k, r.Status.Annotations[k])
		}
	}
	if verbose {
		fmt.Printf("Info: %v\n", r.GetInfo())
	}

	return nil
}

// ListPodSandboxes sends a ListPodSandboxRequest to the server, and parses
// the returned ListPodSandboxResponse.
func ListPodSandboxes(client pb.RuntimeServiceClient, opts listOptions) error {
	filter := &pb.PodSandboxFilter{}
	if opts.id != "" {
		filter.Id = opts.id
	}
	if opts.state != "" {
		st := &pb.PodSandboxStateValue{}
		st.State = pb.PodSandboxState_SANDBOX_NOTREADY
		switch strings.ToLower(opts.state) {
		case "ready":
			st.State = pb.PodSandboxState_SANDBOX_READY
			filter.State = st
		case "notready":
			st.State = pb.PodSandboxState_SANDBOX_NOTREADY
			filter.State = st
		default:
			log.Fatalf("--state should be ready or notready")
		}
	}
	if opts.labels != nil {
		filter.LabelSelector = opts.labels
	}
	request := &pb.ListPodSandboxRequest{
		Filter: filter,
	}
	logrus.Debugf("ListPodSandboxRequest: %v", request)
	r, err := client.ListPodSandbox(context.Background(), request)
	logrus.Debugf("ListPodSandboxResponse: %v", r)
	if err != nil {
		return err
	}
	r.Items = getSandboxesList(r.GetItems(), opts)

	switch opts.output {
	case "json":
		return outputProtobufObjAsJSON(r)
	case "yaml":
		return outputProtobufObjAsJSON(r)
	case "table", "":
	// continue; output will be generated after the switch block ends.
	default:
		return fmt.Errorf("unsupported output format %q", opts.output)
	}

	w := tabwriter.NewWriter(os.Stdout, 20, 1, 3, ' ', 0)
	if !opts.verbose && !opts.quiet {
		fmt.Fprintln(w, "POD ID\tCREATED\tSTATE\tNAME\tNAMESPACE\tATTEMPT")
	}
	for _, pod := range r.Items {
		// Filter by pod name/namespace regular expressions.
		if !matchesRegex(opts.nameRegexp, pod.Metadata.Name) {
			continue
		}
		if !matchesRegex(opts.podNamespaceRegexp, pod.Metadata.Namespace) {
			continue
		}

		if opts.quiet {
			fmt.Printf("%s\n", pod.Id)
			continue
		}
		if !opts.verbose {
			createdAt := time.Unix(0, pod.CreatedAt)
			ctm := units.HumanDuration(time.Now().UTC().Sub(createdAt)) + " ago"
			id := pod.Id
			if !opts.noTrunc {
				id = getTruncatedID(id, "")
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%d\n",
				id, ctm, convertPodState(pod.State), pod.Metadata.Name, pod.Metadata.Namespace, pod.Metadata.Attempt)
			continue
		}

		fmt.Printf("ID: %s\n", pod.Id)
		if pod.Metadata != nil {
			if pod.Metadata.Name != "" {
				fmt.Printf("Name: %s\n", pod.Metadata.Name)
			}
			if pod.Metadata.Uid != "" {
				fmt.Printf("UID: %s\n", pod.Metadata.Uid)
			}
			if pod.Metadata.Namespace != "" {
				fmt.Printf("Namespace: %s\n", pod.Metadata.Namespace)
			}
			if pod.Metadata.Attempt != 0 {
				fmt.Printf("Attempt: %v\n", pod.Metadata.Attempt)
			}
		}
		fmt.Printf("Status: %s\n", convertPodState(pod.State))
		ctm := time.Unix(0, pod.CreatedAt)
		fmt.Printf("Created: %v\n", ctm)
		if pod.Labels != nil {
			fmt.Println("Labels:")
			for _, k := range getSortedKeys(pod.Labels) {
				fmt.Printf("\t%s -> %s\n", k, pod.Labels[k])
			}
		}
		if pod.Annotations != nil {
			fmt.Println("Annotations:")
			for _, k := range getSortedKeys(pod.Annotations) {
				fmt.Printf("\t%s -> %s\n", k, pod.Annotations[k])
			}
		}
		fmt.Println()
	}

	w.Flush()
	return nil
}

func convertPodState(state pb.PodSandboxState) string {
	switch state {
	case pb.PodSandboxState_SANDBOX_READY:
		return "Ready"
	case pb.PodSandboxState_SANDBOX_NOTREADY:
		return "NotReady"
	default:
		log.Fatalf("Unknown pod state %q", state)
		return ""
	}
}

func getSandboxesList(sandboxesList []*pb.PodSandbox, opts listOptions) []*pb.PodSandbox {
	sort.Sort(sandboxByCreated(sandboxesList))
	n := len(sandboxesList)
	if opts.latest {
		n = 1
	}
	if opts.last > 0 {
		n = opts.last
	}
	n = func(a, b int) int {
		if a < b {
			return a
		}
		return b
	}(n, len(sandboxesList))

	return sandboxesList[:n]
}
