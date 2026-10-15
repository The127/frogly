# a project to plant in
POST {{frogly}}/api/projects
Authorization: Bearer {{token}}
Content-Type: application/json
{"name": "Hurl garden"}
HTTP 201
[Captures]
project: jsonpath "$.id"

# a new garden is empty
GET {{frogly}}/api/projects/{{project}}/ideas
Authorization: Bearer {{token}}
HTTP 200
[Asserts]
jsonpath "$" count == 0

# planting an idea answers with it, the text trimmed
POST {{frogly}}/api/projects/{{project}}/ideas
Authorization: Bearer {{token}}
Content-Type: application/json
{"text": "  search should find ideas by any word  "}
HTTP 201
[Captures]
idea: jsonpath "$.id"
[Asserts]
header "Content-Type" == "application/json"
header "Cache-Control" == "no-store"
jsonpath "$.text" == "search should find ideas by any word"
jsonpath "$.plantedBy" == "frogly-tests"
jsonpath "$.plantedAt" isIsoDate

# and the garden lists it
GET {{frogly}}/api/projects/{{project}}/ideas
Authorization: Bearer {{token}}
HTTP 200
[Asserts]
jsonpath "$" count == 1
jsonpath "$[0].id" == "{{idea}}"
jsonpath "$[0].text" == "search should find ideas by any word"

# a text frogly can't use says what's wrong. The other refusals are
# tested in internal/server, this checks the problem on the wire.
POST {{frogly}}/api/projects/{{project}}/ideas
Authorization: Bearer {{token}}
Content-Type: application/json
{"text": "   "}
HTTP 400
[Asserts]
header "Content-Type" == "application/problem+json"
jsonpath "$.detail" == "an idea needs a text"

# a project that doesn't exist is not found
GET {{frogly}}/api/projects/01a14180-d957-7cc5-af29-7f1fe18d89bb/ideas
Authorization: Bearer {{token}}
HTTP 404
[Asserts]
header "Content-Type" == "application/problem+json"

# without a token, frogly asks for one
GET {{frogly}}/api/projects/{{project}}/ideas
HTTP 401
[Asserts]
header "WWW-Authenticate" == "Bearer"
