# K2S
Yet another lightweight version for K8S, and even lighter than K3S.

K2S is base on K3S(https://github.com/rancher/k3s/) from Rancher, but there are some different design goals:
1. focus on ARM64 only, code will be further reduced and fully optimized;
2. better support for lightweight Virtual Machine, novel Container Runtime, and Unikernel;
3. use different networking and storage mechanism from K3S;
4. mainly target at Open Hardware ARM Platforms like 96Boards，Rasperry Pi for IoT/Edge/Microserver/Devops/HCI...    
   It could also work well on our TuobaOS.
