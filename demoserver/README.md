# demoserver

This is a demo server, using the available libraries in this repository. It is also meant to show some design patterns, not suitable for library implementation, but still useful as templates for a server implementation.


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
        	role database
      -tlsCert string
        	server_config/cert.pem (generate with golang's crypto/tls/generate_cert.go) (default disabled)
      -tlsKey string
        	server_config/key.pem (generate with golang's crypto/tls/generate_cert.go) (default disabled)
      -u string
        	user database
    
