#! /bin/bash

# pull down and build the code
sudo rm -rf amazon-ecs-agent
mkdir amazon-ecs-agent
git clone --depth 1 --recurse-submodules --shallow-submodules --branch poc/vpc-bridge https://github.com/YashdalfTheGray/amazon-ecs-agent amazon-ecs-agent
cd amazon-ecs-agent
make cni-plugins
cd ..
rm -rf plugins
mkdir plugins
cp amazon-ecs-agent/out/cni-plugins/* plugins
rm -rf cni-amd64
mkdir cni-amd64
wget https://github.com/containernetworking/plugins/releases/download/v0.9.1/cni-plugins-linux-amd64-v0.9.1.tgz
tar -xvzf cni-plugins-linux-amd64-v0.9.1.tgz -C cni-amd64
cp cni-amd64/host-local plugins/host-local
rm cni-plugins-linux-amd64-v0.9.1.tgz