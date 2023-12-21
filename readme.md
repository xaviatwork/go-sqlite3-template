# Objetivo

En mi empeño de aprender a programar en Go, la mayoría de las aplicaciones que desarrollo son *pruebas de concepto*. El objetivo es entender una determinada *buena práctica*, implementar una aplicación *típica*, un patrón *idiomático*...
Es decir, la funcionalidad no suele ser un requerimiento.

Sin embargo, en esta entrada me centro en el desarrollo de un *paquete* que implemente las funciones CRUD (*create*, *read*, *update* y *delete*) de registros en una base de datos SQL. En este caso, SQLite.
Hasta ahora, el uso de bases de datos lo veía como un *engorro*, precisamente por lo simples que suelen ser las aplicaciones que desarrollo. La idea de tener que añadir contenedores adicionales para la base de datos me frenaba...
Hasta que me di cuenta de que podía obtener la misma funcionalidad usando SQLite.

Como *objetivo secundario*, quiero practicar TDD (*test driven development*) e irlo incorporando a mi manera de desarrollar desde el principio en futuros proyectos.

## El paquete

La idea es crear un paquete que implemente las operaciones CRUD en SQLite3.
Esas funciones se usarán como parte de una aplicación que usa la base de datos para gestionar usuarios.
Estos usuarios se identifican mediante una dirección de correo electrónico y una contraseña.

## Las funciones

Además de las funciones CRUD, necesitamos una función que nos permita *conectar* con la base de datos.

La función de *conexión* debe crear la tabla donde se almacenarán los usuarios (si no existe).

Para las operaciones de lectura, actualización y eliminación de un usuario, necesitamos que la base de datos contenga, al menos, un usuario.
Crearemos también una función que nos facilite el trabajo y que inserte un usuario generado aleatoriamente en la base de datos.
Al final de cada test, eliminaremos el usuario insertado para realizar el test, de manera que cada test encuentre la base de datos siempre en el mismo estado.

## Conexión con la base de datos

Para conectar con la base de datos, usarmeos un DSN (*data source name*). En SQLite el DSN empieza por `file:`, indicando la ruta al fichero que contiene la base de datos.
Podemos usar también el *nombre especial* `:memory:` para generar una base de datos en memoria, que se elimina automáticamente al cerrar la conexión.

Aunque es habitual usar una variable global para almacenar el puntero a la base de datos, prefiero usar un `struct`, con el campo `cnx` no exportado y usar un *constructor* para obtener el puntero.

```go
type Database struct {
    cnx *sql.DB
}
```

## ¿Métodos o funciones?

Empezamos con la función de conexión a la base de datos.
Lo primero que tenemos que decidir es si queremos una función o un método (asociado a `Database`).

Es decir:

```go
// Function
func Delete(u *User, db *Database) error {}
// Method
func (db *Database) Delete(u *User) error {}
```

En este caso, creo que como todas las operaciones giran alrededor de las acciones que se realizan en la base de datos, es mejor usar *métodos* y evitar tener que estar pasando la base de datos como parámetro constantemente.

## Constructor de la conexión con la base de datos

La función de conexión con la base de datos tendrá un doble objetivo:

- conectar con la base de datos
- asegurar la existencia de la tabla `users`

Empezamos con la primera tarea:

Si la conexión con la base de datos tiene éxito, se devuelve `*Database` y `nil`.
Si no, se devuelve el error que se haya producido y `*Database` es `nil`.

Definimos el test como:

> Definimos los tests en un paquete separado llamado `gosqlite3_test`

```go
func TestConnect(t *testing.T) {
    dsn := "file::memory:"
    _, err := gosqlite3.Connect(dsn)
    if err != nil {
        t.Errorf("failed to connect to DB %q: %s", dsn, err.Error())
    }
}
```

La mínima cantidad de código que hace que el test compile es:

```go
func Connect(dsn string) (*Database, error) {
    return nil, nil
}
```

Ahora *refactorizamos* para que `Connect` haga *algo útil*.

Para conectar con la base de datos, como usamos SQLite, tenemos que importar un *driver* específico.

En mi caso uso [github.com/mattn/go-sqlite3](https://pkg.go.dev/github.com/mattn/go-sqlite3).

```console
$ go get github.com/mattn/go-sqlite3
go: downloading github.com/mattn/go-sqlite3 v1.14.19
go: added github.com/mattn/go-sqlite3 v1.14.19
```

También lo añado al fichero `gosqlite3.go`:

```go
import (
    "database/sql"

    _ "github.com/mattn/go-sqlite3"
)
```

Y validamos que tras hacer cambios, el test sigue pasando:

```go
func Connect(dsn string) (*Database, error) {
    driverName := "sqlite3"
    conn, err := sql.Open(driverName, dsn)
    if err != nil {
        return &Database{}, err
    }
    return &Database{cnx: conn}, nil
}
```

Este test sólo valida el caso en el que `Connect` tiene éxito (es decir, `err==nil`).

Para validar el caso en el que la función `Connect` falle, he indicado en el `dsn` una ruta en la carpeta `/root`, de manera que la aplicación no pueda crear el fichero y genere un error.

> Inicialmente el test seguía siendo exitoso, aunque el fichero no se estaba creando. Al parecer, hasta que no se realiza alguna acción contra la base de datos, no se crea el fichero... Para evitar este problema, he usado la función [Ping](https://pkg.go.dev/database/sql#DB.Ping).

En primer lugar, introducimos una verificación de que la conexión con la base de datos ha sido existosa mediante `Ping`:

```go
func Connect(dsn string) (*Database, error) {
    driverName := "sqlite3"
    conn, err := sql.Open(driverName, dsn)
    if err != nil {
        return &Database{}, err
    }
    db := &Database{cnx: conn}
    if err := db.cnx.Ping(); err != nil {
        return &Database{}, err
    }
    return &Database{cnx: conn}, nil
}
```

Por otro lado, en el propio test, vamos a verificar los posibles casos. Definimos un *struct* con los *test cases* e iteramos sobre los diferentes escenarios:

```go
func TestConnect(t *testing.T) {
    type testCase struct {
        description string
        input       string
        output      error
    }
    testcase := []testCase{
        {description: "connection succeeds", input: "file::memory:", output: nil},
        {description: "connection fails", input: "file:/root/db4test.db", output: sqlite3.ErrCantOpen},
    }
    for _, tc := range testcase {
        _, err := gosqlite3.Connect(tc.input)
        if err != nil {
            if sqlite3Err := err.(sqlite3.Error); sqlite3Err.Code != tc.output {
                t.Errorf("%s (for %q): %s", tc.description, tc.input, err.Error())
            }
        }
    }
}
```

## Crear la tabla de `users`

La función `Connect` también debe validar que la tabla `users` existe en la base de datos con la que se ha conectado.

La creación de la tabla se realiza mediante la *query*:

```console
CREATE TABLE
    IF NOT EXISTS
        $TABLENAME (
            email TEXT PRIMARY KEY,
            password TEXT NOT NULL
        )
```

Actualizamos `Connect` para ejecutar la *query* de creación de la tabla.

```go
func Connect(dsn string) (*Database, error) {
    driverName := "sqlite3"
    tableName := "users"
    sqlCreateTable := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (email TEXT PRIMARY KEY, password TEXT NOT NULL);", tableName)
    conn, err := sql.Open(driverName, dsn)
    if err != nil {
        return &Database{}, err
    }
    db := &Database{cnx: conn}
    if err := db.cnx.Ping(); err != nil {
        return &Database{}, err
    }
    _, err = db.cnx.Exec(sqlCreateTable)
    if err != nil {
        return &Database{}, err
    }
    return &Database{cnx: conn}, nil
}
```

Para validar si la tabla se crea correctamente, modificamos la cadena de conexión del primer test para crear una base de datos en disco:

```go
// ...
{description: "connection succeeds", input: "file:db4test.db", output: nil},
// ...
```

Ejecutamos el test y validamos que sigue siendo exitoso; conectamos a la base de datos para validar que la tabla `users` se ha creado:

```console
$ sqlite3 gosqlite3/db4test.db 
SQLite version 3.40.1 2022-12-28 14:03:47
Enter ".help" for usage hints.
sqlite> .tables
users
```
