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