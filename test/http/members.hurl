# a project to add to
POST {{frogly}}/api/projects
Authorization: Bearer {{token}}
Content-Type: application/json
{"name": "Hurl garden", "visibility": "private"}
HTTP 201
[Captures]
project: jsonpath "$.id"

# someone frogly doesn't know says what's wrong
POST {{frogly}}/api/projects/{{project}}/members
Authorization: Bearer {{token}}
Content-Type: application/json
{"userId": "01a14180-d957-7cc5-af29-7f1fe18d89bb"}
HTTP 400
[Asserts]
header "Content-Type" == "application/problem+json"
jsonpath "$.detail" == "nobody with that ID uses frogly"

# without a token, frogly asks for one
POST {{frogly}}/api/projects/{{project}}/members
Content-Type: application/json
{"userId": "01a14180-d957-7cc5-af29-7f1fe18d89bb"}
HTTP 401
[Asserts]
header "WWW-Authenticate" == "Bearer"
