Libraries for user and role management.

# userdb

A simple user database, saved on disk as a text file.

Tab-separated file format:

1. username
2. argon2 hashed password

In some cases, the file may also contain database internal instructions, such as DELETE followed by a username.

Sample file:

     angela	$argon2id$v=19$m=65536,t=3,p=2$9e8pod5QJIVEXND92rjxnQ$IX0Oq3bNhfq4K9lZDUlIfLwH0ZAE0pDv/q55xi8Yasc
     james	$argon2id$v=19$m=65536,t=3,p=2$U4sN8dpRsI2TTEqImgWLig$VEhw7GHD0O8cW0Pl+CB26OHfIpbloBtfj/BsbFesU8c


# roles

A simple database of roles/permissions, saved on disk as a text file.

Tab-separated file format:

1. role name
2. comma-separated list of users

In some cases, the file may also contain database internal instructions, such as DELETE followed by a role name.

 Sample file:
 
    member	angela james
    admin	james
