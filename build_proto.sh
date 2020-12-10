# C#
protoc \
  -I ~/go/src/github.com/DanTulovsky/pepper-poker-v2/proto \
  --csharp_out="/Users/dant/Unity Local/Projects/pepper-poker/Assets/Scripts/Generated/" \
  --grpc_out="/Users/dant/Unity Local/Projects/pepper-poker/Assets/Scripts/Generated/" \
  --plugin=protoc-gen-grpc=/usr/local/bin/grpc_csharp_plugin \
  poker.proto

# Go
protoc \
  -I ~/go/src/github.com/DanTulovsky/pepper-poker-v2/proto \
  --go_out=plugins=grpc:. \
  --go_opt=module=github.com/DanTuovsky/pepper-poker-v2 \
  poker.proto
