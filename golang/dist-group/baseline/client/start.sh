for i in 1 2 3 4; do
  echo "go run client.go --host localhost --port 800$i --file /Users/alex.gaas/Desktop/go/dist-group/base/data/phones_data.csv"
  # run script
  go run client.go --host localhost --port 800"$i" --file /Users/alex.gaas/Desktop/go/dist-group/base/data/phones_data.csv
done