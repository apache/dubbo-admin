LOOP_SIZE=60
i=0

while [[ $i -lt LOOP_SIZE ]]; do
	status_code=$(curl --write-out %{http_code} --silent --output /dev/null http://admin:8080)

  if [[ "$status_code" -eq 200 ]] ; then
    echo "Tests passed!"
    exit 0
  else
    curl -v http://admin:8080
    echo "status is incorrect, waiting for next turn"
  fi
	sleep 5
	i=$i+1
done

echo "Tests failed!"
exit 1