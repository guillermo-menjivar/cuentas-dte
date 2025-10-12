curl -X POST http://167.172.230.154:8113/firmardocumento/ \
  -H "Content-Type: application/json" \
  -d @test_simple_firmador.json | jq .
  #-d @test_firmador.json | jq .
