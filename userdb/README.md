Libraries for user and role management.

# userdb

A simple user database, saved on disk as a text file.

File format:

 1. username &lt;TAB&gt; argon2 hashed password
 2. DELETE &lt;TAB&gt; username

Sample file:

     angela	$argon2id$v=19$m=65536,t=3,p=2$9e8pod5QJIVEXND92rjxnQ$IX0Oq3bNhfq4K9lZDUlIfLwH0ZAE0pDv/q55xi8Yasc
     james	$argon2id$v=19$m=65536,t=3,p=2$U4sN8dpRsI2TTEqImgWLig$VEhw7GHD0O8cW0Pl+CB26OHfIpbloBtfj/BsbFesU8c


# roles

A simple database of roles/permissions, saved on disk as a text file.

File format:

 1. rolename &lt;TAB&gt; users (comma-separated)
 2. DELETE &lt;TAb&gt; rolename
 
 Sample file:
 
    member	angela james
    admin	james
