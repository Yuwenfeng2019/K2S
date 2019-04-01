# K2S
Yet another lightweight version for K8S, and even lighter than K3S(https://github.com/rancher/k3s/).

K2S is base on K3S(https://github.com/rancher/k3s/) from Rancher, but there are some different design goals:
1. focus on ARM64 only, so corresponding code will be further reduced and fully optimized;
2. better support for lightweight Virtual Machine, novel Container Runtimes, and so on;
3. use different networking and storage mechanism from K3S;
4. mainly target at Open Hardware ARM Platforms like 96Boards，Rasperry Pi and so on for 
   IoT/Edge/Microserver/Devops/HCI/AI/Blockchain...    
   It could also work well on our TuobaOS;
5. reduce the third party dependencies(includes Google & Rancher) as much as possible, which means the code base
   of K2S should be self-contained as far as possible.

Note:
1) the initial code was forked from K3S, but we have made some changes and will continue to modify it to achieve the
   design goals of K2S that listed above;
2) the code of K2S will limited to User Space.


