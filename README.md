# K2S
Yet another lightweight version for K8S, and even lighter than K3S.

K2S is base on K3S(https://github.com/rancher/k3s/) from Rancher, but there are some different design goals:
1. focus on ARM architecture for further optimization(may add RISC-V support in the future);
2. better support for Micro Virtual Machine and novel Container Runtimes such like Firecraker & Kata Container;
3. different networking and storage solutions:
   * integrates Network Service Mesh(https://github.com/networkservicemesh/) which allows for a separate
       (and performant when required) data path from the CNI
   * enhanced Sqlite3 and Dqlite implementation
4. mainly target at Open Hardware Platforms like 96Boards/Raspberry Pi, Lattepanda, and so on for
   IoT/Edge/Microserver/DevOps/HCI/AI/Blockchain...
   In addition, it could also work well on our TuobaOS;
5. reduce the third party dependencies(includes Google & Rancher) as much as possible, which means the code base
   of K2S should be self-contained as far as possible.

Note:
1) the initial code was forked from K3S, but we have made some changes and will continue to modify it to achieve the
   design goals of K2S that listed above;
2) the code base of K2S will be limited to User Space.

Sync with K3S:
1) K3S v0.3.0 added Air-Gap support
2) currently v1.17.3+k3s1
