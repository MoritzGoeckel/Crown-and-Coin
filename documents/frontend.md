- The main should start a webserver serving static content from the `static` folder. In that folder we need an index.html with a js and a css file
- The main needs to have some kind of user management. The user have a name and a secret. When a user opens the website it should ask it for a name and secret and then send it to register on the server
- The main should provide a websocket server where the web clients can connect. The websocket receives messages from the client in json format.

The websocket messages follow that patter:
{"user": "", "secret": "", "payload": {}}

- The server should keep track of the registered users and their secrets. When the server starts it should output the secret of the admin user. The admin user should exist from the beginning.
- If a message from a user has the wrong secret, it should be discarded and a error message printed and returned to the client
- The server should start a game using the engine. 
- Check test\scenarios.json to understand the possible messages, create a verification method to check if the "user" is allowed to send this message. The admin user can send any message. The response of the engine should be send back via websocket to the client. Everyone can request get_state
