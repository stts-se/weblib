# demoserver

This is a demo web server, using the available libraries in this repository. It is also meant to show some design patterns, not suitable for library implementation, but still useful as templates for a server implementation.

## Usage

    $ ./demoserver -help
    Usage of ./demoserver:
      -h	print usage and exit
      -host string
        	server host (default "127.0.0.1")
      -key string
        	server key file for session cookies (default "server_config/serverkey")
      -port int
        	server port (default 7932)
      -r string
        	role database (required)
      -tlsCert string
        	server_config/cert.pem (generate with golang's crypto/tls/generate_cert.go) (default disabled)
      -tlsKey string
        	server_config/key.pem (generate with golang's crypto/tls/generate_cert.go) (default disabled)
      -u string
        	user database (required)
    

## Example usage

    $ ./demoserver -u userdb.txt -r roles.txt 
    2019/05/21 17:29:44 Read locale en from file i18n/en.properties
    2019/05/21 17:29:44 Read locale sv from file i18n/sv.properties
    2019/05/21 17:29:44 Created user database userdb.txt
    2019/05/21 17:29:44 Created role database roles.txt
    2019/05/21 17:29:44 Getting ready to start server on http://127.0.0.1:7932
    2019/05/21 17:29:44 Server up and running on http://127.0.0.1:7932
