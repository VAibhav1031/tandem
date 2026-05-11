package cookies

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"os"

	_ "modernc.org/sqlite"
)

func copyfile(dest_name string, src string) (*os.File, error) {
	//******************COPYING_COOKIE_FILE**********

	// path of the Cookie file

	cookie_org, err := os.Open(src)
	if err != nil {
		panic(err)
	}
	defer cookie_org.Close()

	temp_cookie_path, err := os.CreateTemp("", dest_name+"-*")
	//need to create the temp fiel and all shit
	if err != nil {
		panic(err)
	}

	_, err = io.Copy(temp_cookie_path, cookie_org)

	return temp_cookie_path, err
}

func CookieSolver() string {

	// we need to open the  shit

	home_dir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	cookie_path := home_dir + "/.config/chromium/Default/Cookies"
	temp_file_path, err := copyfile("Cookies", cookie_path)

	defer os.Remove(temp_file_path.Name())
	defer temp_file_path.Close()
	if err != nil {
		log.Fatal("Creation of Temp Failed", err)
		return ""
	}

	db, err := sql.Open("sqlite", temp_file_path.Name())
	if err != nil {
		fmt.Printf("Error Occurred %v", err)
		return ""
	}
	defer db.Close()

	// Ping to the db to check if connection working well or not
	if err := db.Ping(); err != nil {
		fmt.Println("Failed to connect:", err)
	}

	// rows, err := db.Query("SELECT name FROM sqlite_master ;")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// defer rows.Close()
	//
	// fmt.Println("Tables in database:")
	// for rows.Next() {
	// 	var tableName string
	// 	if err := rows.Scan(&tableName); err != nil {
	// 		log.Fatal(err)
	// 	}
	// 	fmt.Printf("- %s\n", tableName)
	// }
	//
	// get the latest one
	rows, err := db.Query("Select encrypted_value from cookies where name = 'cf_clearance' and host_key = '.testfile.org' order by creation_utc desc limit 1;")

	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	var encrypted_value string
	for rows.Next() {
		if err := rows.Scan(&encrypted_value); err != nil {
			log.Fatal(err)
		}
		//fmt.Printf("- %s\n", row_)
	}

	key := DbusKeyGetter()
	cookie, err := decryptCookie([]byte(encrypted_value), key)
	if err != nil {
		log.Fatal("There is no Cookie, Currently")
		return ""
	}
	return cookie

}
