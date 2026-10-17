# a public project of your own
POST {{frogly}}/api/projects
Authorization: Bearer {{token}}
Content-Type: application/json
{"name": "Hurl fair", "visibility": "public"}
HTTP 201
[Captures]
id: jsonpath "$.id"

# isn't among the public projects of others
GET {{frogly}}/api/projects/public
Authorization: Bearer {{token}}
HTTP 200
[Asserts]
header "Content-Type" == "application/json"
header "Cache-Control" == "no-store"
jsonpath "$" isCollection
jsonpath "$[?(@.id == '{{id}}')]" not exists

# without a token, frogly asks for one
GET {{frogly}}/api/projects/public
HTTP 401
[Asserts]
header "WWW-Authenticate" == "Bearer"
