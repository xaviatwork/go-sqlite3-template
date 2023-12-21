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
