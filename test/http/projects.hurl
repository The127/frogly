# creating a project answers with it, the name trimmed
POST {{frogly}}/api/projects
Authorization: Bearer {{token}}
Content-Type: application/json
{"name": "  Hurl garden  ", "visibility": "private"}
HTTP 201
[Captures]
id: jsonpath "$.id"
[Asserts]
header "Content-Type" == "application/json"
header "Cache-Control" == "no-store"
jsonpath "$.name" == "Hurl garden"
jsonpath "$.visibility" == "private"
jsonpath "$.joining" == "closed"

# and lists it among your projects
GET {{frogly}}/api/projects
Authorization: Bearer {{token}}
HTTP 200
[Asserts]
jsonpath "$[?(@.id == '{{id}}')].name" == "Hurl garden"

# and answers with it on its own
GET {{frogly}}/api/projects/{{id}}
Authorization: Bearer {{token}}
HTTP 200
[Asserts]
header "Content-Type" == "application/json"
jsonpath "$.id" == "{{id}}"
jsonpath "$.name" == "Hurl garden"

# a project that doesn't exist is not found
GET {{frogly}}/api/projects/01a14180-d957-7cc5-af29-7f1fe18d89bb
Authorization: Bearer {{token}}
HTTP 404
[Asserts]
header "Content-Type" == "application/problem+json"

# a name frogly can't use says what's wrong. The other refusals are
# tested in internal/server, this checks the problem on the wire.
POST {{frogly}}/api/projects
Authorization: Bearer {{token}}
Content-Type: application/json
{"name": "   ", "visibility": "private"}
HTTP 400
[Asserts]
header "Content-Type" == "application/problem+json"
jsonpath "$.detail" == "a project needs a name"

# without a token, frogly asks for one
GET {{frogly}}/api/projects
HTTP 401
[Asserts]
header "WWW-Authenticate" == "Bearer"

# projects can't be deleted yet
DELETE {{frogly}}/api/projects
Authorization: Bearer {{token}}
HTTP 405
[Asserts]
header "Allow" == "GET, HEAD, POST"
