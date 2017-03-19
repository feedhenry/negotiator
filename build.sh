set -e
cd cmd/negotiator
go build .
cd ..
cd ..
imagehash=`docker build -q .`
docker tag $imagehash rhmap/negotiator:0.0.4
oc deploy --latest negotiator 

