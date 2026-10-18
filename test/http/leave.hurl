# a project of your own to leave
POST {{frogly}}/api/projects
Authorization: Bearer {{token}}
Content-Type: application/json
{"name": "Hurl shed", "visibility": "private"}
HTTP 201
[Captures]
id: jsonpath "$.id"

# leaving answers with nothing
POST {{frogly}}/api/projects/{{id}}/leave
Authorization: Bearer {{token}}
HTTP 204

# and the project is gone for you, so leaving again is not found
POST {{frogly}}/api/projects/{{id}}/leave
Authorization: Bearer {{token}}
HTTP 404
[Asserts]
header "Content-Type" == "application/problem+json"
