# K2S
Yet another lightweight version for K8S, and even lighter than K3S(https://github.com/rancher/k3s/).

K2S is base on K3S(https://github.com/rancher/k3s/) from Rancher, but there are some different design goals:
1. focus on 64-bit system only(currently X64 & ARM64, and RISC-V 64 in the future), so corresponding code will 
   be further reduced and fully optimized;
2. better support for lightweight Virtual Machine/novel Container Runtimes such like Kata Container & Firecraker;
3. use different networking and storage mechanism from K3S;
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
2) currently 0.4.0-RC2
