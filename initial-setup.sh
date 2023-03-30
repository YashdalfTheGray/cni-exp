#! /bin/bash

# pull down and build the code
rm -rf amazon-vpc-cni-plugins
curl -sL https://github.com/aws/amazon-vpc-cni-plugins/archive/master.tar.gz | tar xzv
mv amazon-vpc-cni-plugins-master amazon-vpc-cni-plugins
cd amazon-vpc-cni-plugins
make vpc-bridge
cd ..