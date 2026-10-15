# creating a project answers with it, the name trimmed
POST {{frogly}}/api/projects
Authorization: Bearer {{token}}
Content-Type: application/json
{"name": "  Hurl garden  "}
HTTP 201
[Captures]
id: jsonpath "$.id"
[Asserts]
header "Content-Type" == "application/json"
header "Cache-Control" == "no-store"
jsonpath "$.name" == "Hurl garden"

# and lists it among your projects
GET {{frogly}}/api/projects
Authorization: Bearer {{token}}
HTTP 200
[Asserts]
jsonpath "$[?(@.id == '{{id}}')].name" == "Hurl garden"

# a name frogly can't use says what's wrong
POST {{frogly}}/api/projects
Authorization: Bearer {{token}}
Content-Type: application/json
{"name": "   "}
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
