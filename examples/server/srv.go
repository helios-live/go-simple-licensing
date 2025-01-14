package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/rotisserie/eris"

	Licensing "github.com/ideatocode/go-simple-licensing"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/pelletier/go-toml"
)

var (
	PORT string
	SSL  bool

	ALLOWFWR string
	HOST     string
	DBPORT   string
	DATABASE string
	USERNAME string
	PASSWORD string

	db  *sql.DB
	err error

	ConfigRaw string = `[server]
port = "{PORT}"
ssl = {SSL}

[database]
host = "{HOST}"
db = "{DB}"
username = "{USERNAME}"
password = "{PASSWORD}"`
)

func loadConfig() {
	config, err := toml.LoadFile("config.toml")
	if err != nil {
		fmt.Println("[ERROR] Could not load config.toml!")
		os.Exit(0)
	} else {
		PORT = config.Get("server.port").(string)
		SSL = config.Get("server.ssl").(bool)
		ALLOWFWR = config.Get("server.forwarding_from").(string)

		HOST = config.Get("database.host").(string)
		DATABASE = config.Get("database.db").(string)
		USERNAME = config.Get("database.username").(string)
		PASSWORD = config.Get("database.password").(string)
	}
}

func convertInt(input string) (bool, int) {
	i, err := strconv.Atoi(input)
	if err != nil {
		return false, 0
	}
	return true, i
}

func convertBool(input string) (bool, bool) {
	i, err := strconv.ParseBool(input)
	if err != nil {
		return false, false
	}
	return true, i
}

func randomString(n int) string {
	var letterRunes = []rune("1234567890ABCDEFGHIJKLMNOPQRSTUVWXYZ")

	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func setup() {
	fmt.Println("Server Setup")
	fmt.Println("")
	var tmpconfig string = ConfigRaw

SetupPort:
	fmt.Print("Port # to Run Server On: ")
	scan := bufio.NewScanner(os.Stdin)
	scan.Scan()
	a, _ := convertInt(scan.Text())
	if !a {
		fmt.Println("Port can only be numbers")
		goto SetupPort
	}
	tmpconfig = strings.Replace(tmpconfig, "{PORT}", scan.Text(), -1)
SetupSSL:
	fmt.Print("Use SSL (true/false): ")
	scan = bufio.NewScanner(os.Stdin)
	scan.Scan()
	c, _ := convertBool(scan.Text())
	if !c {
		fmt.Println("Must be true or false")
		goto SetupSSL
	}
	tmpconfig = strings.Replace(tmpconfig, "{SSL}", scan.Text(), -1)

	fmt.Print("SQL Database Host (127.0.0.1:3306): ")
	scan = bufio.NewScanner(os.Stdin)
	scan.Scan()
	tmpconfig = strings.Replace(tmpconfig, "{HOST}", scan.Text(), -1)

	fmt.Print("SQL Database: ")
	scan = bufio.NewScanner(os.Stdin)
	scan.Scan()
	tmpconfig = strings.Replace(tmpconfig, "{DB}", scan.Text(), -1)

	fmt.Print("SQL Database Username: ")
	scan = bufio.NewScanner(os.Stdin)
	scan.Scan()
	tmpconfig = strings.Replace(tmpconfig, "{USERNAME}", scan.Text(), -1)

	fmt.Print("SQL Database Password: ")
	scan = bufio.NewScanner(os.Stdin)
	scan.Scan()
	tmpconfig = strings.Replace(tmpconfig, "{PASSWORD}", scan.Text(), -1)

	fmt.Println("")
	fmt.Println("Saving config.toml file...")
	d1 := []byte(tmpconfig)
	err := ioutil.WriteFile("config.toml", d1, 0644)
	if err != nil {
		fmt.Println("There was an error creating the config file, please do it manually.")
	}
	if Licensing.CheckFileExist("config.toml") {
		fmt.Println("Setup Finished.")
	}
}

func indexHandler(response http.ResponseWriter, request *http.Request) {
	fmt.Fprintf(response, "Go Simple Licensing Server")
}

func checkHandler(response http.ResponseWriter, request *http.Request) {
	request.ParseForm()
	decrypted := request.FormValue("license")
	var tmpexp string

	if err != nil {
		response.WriteHeader(http.StatusBadRequest)
	}
	err = db.QueryRow("SELECT expiration FROM licenses WHERE license='" + decrypted + "'").Scan(&tmpexp)
	if err == sql.ErrNoRows { //No License for Key found
		fmt.Fprintf(response, "Bad.")
	} else { //Check Expiration date

		// check IP address
		ip := strings.Split(request.RemoteAddr, ":")[0]

		fwr := request.Header.Get("X-Forwarded-For")
		if ALLOWFWR == ip && fwr != "" {
			ip = fwr
		}

		if !checkIP(decrypted, ip) {
			fmt.Println("Bad IP.")
			return
		}

		_, err := db.Exec("UPDATE licenses SET ip='" + ip + "' WHERE license='" + decrypted + "'")
		if err != nil {
			fmt.Println(err)
		}
		t, err := time.Parse("2006-01-02", tmpexp)

		if err != nil {
			fmt.Println(eris.Wrap(err, "ERROR: SQL Table Date no Correct Format"))
		}
		t2, _ := time.Parse("2006-01-02", time.Now().Format("2006-01-02"))
		if t.After(t2) {
			fmt.Fprintf(response, "Good")
		} else {
			fmt.Fprintf(response, "Expired")
		}
	}
}

func checkIP(license, ip string) bool {
	var tmpip string
	err := db.QueryRow("SELECT ip FROM licenses WHERE license='" + license + "'").Scan(&tmpip)
	if err == sql.ErrNoRows { //No License for Key found
		return false
	}

	if tmpip != "" && tmpip != ip {
		return false
	}

	return true

}

func API() {
	router := mux.NewRouter()
	router.HandleFunc("/", indexHandler)
	router.HandleFunc("/check", checkHandler).Methods("POST")
	http.Handle("/", router)
	var err error
	if SSL {
		fmt.Println("Listening on SSL:", PORT)
		err = http.ListenAndServeTLS(":"+PORT, "server.cert.pem", "server.key.pem", nil) //:443
	} else {
		fmt.Println("Listening on HTTP:", PORT)
		err = http.ListenAndServe(":"+PORT, nil)
	}
	if err != nil {
		fmt.Println("Listen Server Error: " + err.Error())
		os.Exit(0)
	}
}

func count() int { //Count Bot Rows
	rows, err := db.Query("SELECT COUNT(*) AS count FROM licenses")
	if err != nil {
		return 0
	}
	var count int

	defer rows.Close()
	for rows.Next() {
		rows.Scan(&count)
	}
	return count
}

func main() {
	fmt.Println("Go Simple Licensing System")

	if SSL {
		if !Licensing.CheckFileExist("server.cert.pem") || !Licensing.CheckFileExist("server.key.pem") {
			fmt.Println("[!] WARNING MAKE SURE YOU HAVE YOUR SSL FILES IN THE SAME DIR [!]")
			os.Exit(0)
		}
	}

	if !Licensing.CheckFileExist("config.toml") {
		setup()
	}

	loadConfig()

	db, err = sql.Open("mysql", USERNAME+":"+PASSWORD+"@tcp("+HOST+")/"+DATABASE)
	if err != nil {
		fmt.Println("[!] ERROR: CHECK MYSQL SETTINGS! [!]")
		os.Exit(0)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		fmt.Println("[!] ERROR: CHECK IF MYSQL SERVER IS ONLINE! [!]")
		os.Exit(0)
	}

	go API()

	for {
	Menu:
		fmt.Println("[Total Licenses]", count())
		fmt.Println("")
		fmt.Print("Console: ")
		scan := bufio.NewScanner(os.Stdin)
		scan.Scan()
		switch scan.Text() {
		case "add":
			var email string
			var expiration string
			var license string

			fmt.Print("License Email: ")
			scan = bufio.NewScanner(os.Stdin)
			scan.Scan()
			email = scan.Text()
		exp:
			fmt.Print("License Expiration (YYYY-MM-DD): ")
			scan = bufio.NewScanner(os.Stdin)
			scan.Scan()
			_, err = time.Parse("2006-01-02", scan.Text())
			if err != nil {
				fmt.Println("Expiration must be in the YYYY-MM-DD Format.")
				goto exp
			}
			expiration = scan.Text()
			fmt.Println("")

			// license = randomString(4) + "-" + randomString(4) + "-" + randomString(4)

			ln, _ := uuid.NewUUID()
			license = ln.String()

			ip := ""
			var tmpemail string
			err := db.QueryRow("SELECT email FROM licenses WHERE license='" + license + "'").Scan(&tmpemail)
			if err == sql.ErrNoRows {
				_, err = db.Exec("INSERT INTO licenses(email, license, expiration, ip) VALUES(?, ?, ?, ?)", email, license, expiration, ip)
				if err != nil {
					fmt.Println("[!] ERROR: UNABLE TO INSERT INTO DATABASE [!]")
					fmt.Println("")
					goto Menu
				}
			} else {
				fmt.Println("License already in database?")
				fmt.Println("License:", license)
				fmt.Println("Email:", tmpemail)
				fmt.Println("")
				goto Menu
			}

			fmt.Println("License Key Generated!")
			fmt.Println("")
			fmt.Println("License Email:", email)
			fmt.Println("License Expiration:", expiration)
			fmt.Println("Save this as license.dat")
			fmt.Println("")
			fmt.Println(license)
			fmt.Println("")

		case "remove":
			fmt.Print("License Email: ")
			scan = bufio.NewScanner(os.Stdin)
			scan.Scan()
			var tmp string
			err = db.QueryRow("SELECT license FROM licenses WHERE email=?", scan.Text()).Scan(&tmp)
			if err == sql.ErrNoRows {
				fmt.Println("[!] ERROR: COULD NOT FIND LICENSE [!]")
				fmt.Println("")
				goto Menu
			} else {
				fmt.Println("License Found:", tmp)
				_ = db.QueryRow("DELETE FROM licenses WHERE email=?", scan.Text())
				fmt.Println("License removed from database.")
				fmt.Println("")
				goto Menu
			}
		case "exit":
			os.Exit(0)
		default:
			fmt.Println("Unknown Command")
		}
	}
}
