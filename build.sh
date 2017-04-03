set -e
cd cmd/negotiator
go build .
cd ..
cd ..
cd cmd/jobs && go build . && cd ../..
imagehash=`docker build -q .`
docker tag $imagehash feedhenry/negotiator:0.0.11
oc deploy --latest negotiator

