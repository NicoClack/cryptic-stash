atlas migrate new "$1" --dir "file://ent/migrate/migrations"
read -p "Press enter once you've written the migration..."
atlas migrate hash --dir "file://ent/migrate/migrations"