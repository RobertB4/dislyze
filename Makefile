initdb:
	DBPASSWORD=password psql -U postgres -d lugia -p 5432 -f ./backend/db/setup.sql
