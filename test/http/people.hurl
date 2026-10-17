# a project to search from
POST {{frogly}}/api/projects
Authorization: Bearer {{token}}
Content-Type: application/json
{"name": "Hurl garden", "visibility": "private"}
HTTP 201
[Captures]
project: jsonpath "$.id"

# the search gives people who could join, never a member like yourself
GET {{frogly}}/api/projects/{{project}}/people?search=frogly
Authorization: Bearer {{token}}
HTTP 200
[Asserts]
header "Content-Type" == "application/json"
header "Cache-Control" == "no-store"
jsonpath "$" isCollection
jsonpath "$[?(@.name == 'frogly-tests')]" not exists

# an empty search says what's missing
GET {{frogly}}/api/projects/{{project}}/people?search=
Authorization: Bearer {{token}}
HTTP 400
[Asserts]
header "Content-Type" == "application/problem+json"
jsonpath "$.detail" == "type part of a name or an email"

# a project that doesn't exist is not found
GET {{frogly}}/api/projects/01a14180-d957-7cc5-af29-7f1fe18d89bb/people?search=frogly
Authorization: Bearer {{token}}
HTTP 404
