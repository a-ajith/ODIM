#!/bin/bash
# (C) Copyright [2020] Hewlett Packard Enterprise Development LP
# 
# Licensed under the Apache License, Version 2.0 (the "License"); you may
# not use this file except in compliance with the License. You may obtain
# a copy of the License at
# 
#     http://www.apache.org/licenses/LICENSE-2.0
# 
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
# WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
# License for the specific language governing permissions and limitations
# under the License.

cd lib-utilities/proto
current_dir=$(pwd)

wget https://github.com/protocolbuffers/protobuf/releases/download/v3.14.0/protoc-3.14.0-linux-x86_64.zip
mkdir proto_files
# if unzip is not installed
sudo apt install unzip
unzip protoc-3.14.0-linux-x86_64.zip -d proto_files
cd proto_files/bin
sudo cp protoc /usr/bin
cd ../include
sudo cp -r google /usr/local/include/
cd
go get github.com/micro/micro/v3/cmd/protoc-gen-micro@master
go install google.golang.org/protobuf/cmd/protoc-gen-go

cd "$current_dir"
pwd

echo "$current_dir"
sub='.proto'
for entry in ./*
do
  if [[ "$entry" == *"$sub" ]];
  then
    dir_name="${entry//.proto}"
    mkdir "$dir_name"
    cp "$entry" "$dir_name"
    cd "$dir_name"
    protoc -I /usr/local/include/ --proto_path=$GOPATH/src:. --micro_out=. --go_out=. "$entry"
    cd ..
  else
    rm -rf "$entry"
  fi
done

