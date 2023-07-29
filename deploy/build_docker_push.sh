#!/bin/bash
passwd=$1
version=$2
echo {passwd} | docker login --username=qinguoyi --password-stdin
docker build -t asyncjob:${version} -f deploy/Dockerfile  .
docker tag asyncjob:${version} qinguoyi/asyncjob:${version}
docker push qinguoyi/asyncjob:${version}
if [ $? -eq 0 ]; then
 echo "push success"
else
 echo "push failed"
fi