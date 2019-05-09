# curl testing

*Login*

    $ curl -i -s -u user:password http://127.0.0.1:7932/auth/login
   
    HTTP/1.1 200 OK
    Set-Cookie: auth-user-session=<COOKIE>; Path=/; Expires=Thu, 16 May 2019 08:51:54 GMT; Max-Age=604800; HttpOnly
    Www-Authenticate: Basic realm="127.0.0.1:7932"
    Date: Thu, 09 May 2019 08:51:54 GMT
    Content-Length: 23
    Content-Type: text/plain; charset=utf-8
     
    Logged in successfully


*Create invitation*

    $ curl -s --cookie "auth-user-session=<COOKIE>" http://127.0.0.1:7932/auth/invite

    Invitation link: http://127.0.0.1:7932/auth/signup/<INVITATION_TOKEN>


*Signup with invitation token*

    $ curl -i -s -u newuser:newpassword http://127.0.0.1:7932/auth/signup/<COOKIE>
