go build . 
mkdir -p ~/.watch-client
cp structure.json ~/.watch-client
[ -d "/path/to/dir" ] && cp .env ~/.watch-client
sudo mv watch /usr/local/bin
echo "Execute 'watch help'"