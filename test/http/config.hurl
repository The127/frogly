# the frontend learns from frogly where to log in
GET {{frogly}}/api/config
HTTP 200
[Asserts]
header "Content-Type" == "application/json"
jsonpath "$.issuer" == "{{issuer}}"
jsonpath "$.clientId" == "frogly"
