CREATE TABLE licenses(
  id INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  email TEXT(120),
  license TEXT(120),
  expiration DATE,
  ip TEXT(120)
)