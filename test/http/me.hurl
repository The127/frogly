# a logged in person gets their name
GET {{frogly}}/api/me
Authorization: Bearer {{token}}
HTTP 200
[Asserts]
header "Content-Type" == "application/json"
header "Cache-Control" == "no-store"
jsonpath "$.name" == "frogly-tests"

# without a token, frogly asks for one
GET {{frogly}}/api/me
HTTP 401
[Asserts]
header "WWW-Authenticate" == "Bearer"
header "Content-Type" == "application/problem+json"
jsonpath "$.status" == 401

# a token frogly didn't get from Keyline is refused
GET {{frogly}}/api/me
Authorization: Bearer forged
HTTP 401
[Asserts]
header "WWW-Authenticate" == "Bearer error=\"invalid_token\""

# the API only reads
POST {{frogly}}/api/me
Authorization: Bearer {{token}}
HTTP 405
[Asserts]
header "Allow" == "GET, HEAD"
