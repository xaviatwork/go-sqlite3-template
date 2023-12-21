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

## Insertar un usuario en la base de datos

Ahora que hemos conectado a la base de datos y que estamos seguros que la tabla `users` se ha creado, ha llegado el momento de insertar nuestro primer usuario.

Generamos las propiedades del nuevo usuario usando [Gofakeit](https://github.com/brianvoe/gofakeit).

En primer lugar, lo descargamos mediante:

```console
go get github.com/brianvoe/gofakeit/v6
```

Empezamos definiendo el *type* `User`:

```go
type User struct {
    Email    string
    Password string
}
```

Para todos los métodos que tenemos que implementar necesitamos que la conexión con la base de datos se haya establecido, que no haya errores. Además, para la mayor parte de los test, es conveniente que ya exista un usuario en la base de datos.

Para evitar tener que repetir todos estos pasos en cada uno de los tests, definimos una función auxiliar `setupDB`.

## Configuración del test: `setupDB`

Si la conexión con la base de datos no puede realizarse, falla la creación de la tabla  `users` o no se puede insertar el usuario, hacemos fallar el test (en vez de devolver el error). Para ello, tenemos que pasar el *test* (`*testing.T`) como parámetro, junto con la cadena de conexión a la base de datos.

Esta función de *inicialización* devuelve la conexión con la base de datos (`*Database`) y el email del usuario insertado (para poder consultarlo, actualizarlo o borrarlo).

Por ahora, dejamos la función de inicialización *a medias*; es decir, de momento, sólo se ocupa de la conexión con la base de datos, pero no se inserta el usuario. Para ello, y por no duplicar el código, esperamos a que hayamos testeado la función `Add` antes de introducirla en `setupDB`.

Como vamos a hacer que `setupDB` falle si se produce cualquier error, el test sólo comprueba si, por cualquier motivo, pese a no haber fallado, no se devuelve alguno de los valores que nos interesan:

```go
func Test_setupDB(t *testing.T) {
    dsn := "file:db4test.db"
    db, email := setupDB(dsn, t)
    if db == nil || email == "" {
        t.Errorf("db setup failed with no error")
    }
}
```

La función `setupDB` es (por ahora):

```go
func setupDB(dsn string, t *testing.T) (*gosqlite3.Database, string) {
    db, err := gosqlite3.Connect(dsn)
    if err != nil {
        t.Errorf("db setup failed: %s", err.Error())
    }
    u := &gosqlite3.User{
        Email:    gofakeit.Email(),
        Password: gofakeit.Password(true, true, true, true, false, 15),
    }
    return db, u.Email
}
```

> Como hemos importado el paquete `gofakeit` tenemos que actualizar las dependencias mediante `go mod tidy`.

## Insertar un usuario en la base de datos (*reprise*)

Mediante `setupDB` tenemos inicializada la conexión con la base de datos, estamos seguros de que la tabla `users` está creada.

Ahora podemos concentrarnos en añadir un usuario.

En este caso, usamos Gofakeit para generar las propiedades del usuario que vamos a insertar en la base de datos. Si no se produce ningún error, consideramos que la inserción ha tenido éxito.

El test es:

```go
func TestAdd(t *testing.T) {
    dsn := "file:db4test.db"
    db, _ := setupDB(dsn, t)
    u := &gosqlite3.User{
        Email:    gofakeit.Email(),
        Password: gofakeit.Password(true, true, true, true, false, 15),
    }
    if err := db.Add(u); err != nil {
        t.Errorf("failed to insert user: %s", err.Error())
    }
}
```

La mínima cantidad de código que satisface el test es:

> Hemos decidido implmentar las operaciones CRUD como métodos del tipo `Database`.

```go
func (db *Database) Add(u *User) error {
    return nil
}
```

Refactorizamos:

> Hemos convertido la variable `tableName` en global dentro del paquete para evitar tener que definirla en cada función. No quiero pasarla como parámetro porque quizás en el futuro me interese obtenerla de una variable de entorno o similar.

```go
func (db *Database) Add(u *User) error {
    tx, err := db.cnx.Begin()
    if err != nil {
        return fmt.Errorf("begin 'add' transaction failed: %w", err)
    }

    sqlInsert := fmt.Sprintf("INSERT INTO %s (email, password) VALUES (?,?)", tableName)
    stmt, err := tx.Prepare(sqlInsert)
    if err != nil {
        return fmt.Errorf("prepare 'add' transaction failed: %w", err)
    }
    defer stmt.Close()

    _, err = stmt.Exec(u.Email, u.Password)
    if err != nil {
        return fmt.Errorf("exec 'add' transaction failed: %w", err)
    }

    tx.Commit()

    return nil
}
```

Tras validar que el test se sigue ejecutando con éxito, conectamos a la base de datos para validar que se ha insertado un usuario en `users`:

```console
$ sqlite3 gosqlite3/db4test.db 
SQLite version 3.40.1 2022-12-28 14:03:47
Enter ".help" for usage hints.
sqlite> select * from users;
marcoratke@graham.com|o|w?O9t|JTO6=_1
sqlite>
```

## Actualizar la función `setupDB`

Antes hemos dejado la función `setupDB` inacabada para desarrollar y testear la función de insertar usuarios antes de usarla.

Ahora añadimos a `setupDB` la capacidad de insertar un usuario de prueba.

```go
func setupDB(dsn string, t *testing.T) (*gosqlite3.Database, string) {
    db, err := gosqlite3.Connect(dsn)
    if err != nil {
        t.Errorf("db setup failed: %s", err.Error())
    }
    u := &gosqlite3.User{
        Email:    gofakeit.Email(),
        Password: gofakeit.Password(true, true, true, true, false, 15),
    }
    err = db.Add(u)
    if err != nil {
        t.Errorf("db setup failed: insert user: %s", err.Error())
    }
    t.Logf("(setupDB) test email: %s", u.Email)
    return db, u.Email
}
```

Hemos añadido unos `t.Logf` que imprimen el valor del *email* del usuario generado (cuando se ejecutan los test en modo *verbose*):

```go
$ go test ./... -v 
=== RUN   TestConnect
--- PASS: TestConnect (0.03s)
=== RUN   TestAdd
    gosqlite3_test.go:65: (setupDB) test email: derekkoelpin@walsh.org
--- PASS: TestAdd (0.01s)
=== RUN   TestDelete
    gosqlite3_test.go:65: (setupDB) test email: zakarymurray@dooley.net
    gosqlite3_test.go:46: (delete): test email: zakarymurray@dooley.net
--- PASS: TestDelete (0.01s)
=== RUN   Test_setupDB
    gosqlite3_test.go:65: (setupDB) test email: hailiekuhn@dare.com
--- PASS: Test_setupDB (0.00s)
PASS
ok      github.com/xaviatwork/gosqlite3/gosqlite3       0.137s
```

Si accedemos a la base de datos, vemos que tenemos todos los usuarios generados por la función `setupDB` excepto para el caso del test del borrado `TestDelete`:

> El usuario adicional lo crea el test `TestAdd`, que no mostraba la dirección de correo del usuario añadido.

```console
$ sqlite3 gosqlite3/db4test.db 
SQLite version 3.40.1 2022-12-28 14:03:47
Enter ".help" for usage hints.
sqlite> .tables
users
sqlite> select * from users;
derekkoelpin@walsh.org|0%TxrB9}zbzn.k+
veronagreenfelder@weimann.info|5)j$?jZ42au0wo}
hailiekuhn@dare.com|__swTu+4kWQTLJ*
sqlite>
```

> He actualizado el test para que se muestre el *email* del usuario añadido.

## Eliminar el usuario de test

Al usar Gofakeit, cada usuario que insertamos en la base de datos tiene valores (pseudo)aleatorios. Sin embargo, para seguir con las buenas prácticas que habíamos indicado al principio, cada test debe encargarse de eliminar el usuario insertado (o modificado).

Vamos a usar [T.Cleanup](https://pkg.go.dev/testing#T.Cleanup) para ejecutar una función al final de cada test (o conjunto de tests).

Para ello, desarrollamos la función de eliminar usuarios y, después, pasaremos a actualizar los tests (y la función de `setupDB`) para elimnar las modificaciones realizadas durante las pruebas.

El test para `Delete`:

```go
func TestDelete(t *testing.T) {
    dsn := "file:db4test.db"
    db, email := setupDB(dsn, t)
    t.Logf("(delete): test email: %s", email)
    if err := db.Delete(email); err != nil {
        t.Errorf("failed to delete user %s, %s", email, err.Error())
    }
}
```

Partimos de:

```go
func (db *Database) Delete(email string) error {
    return nil
}
```

y refactorizamos.

Las accciones para borrar un usuario son muy parecidas a las de crearlo...

```go
func (db *Database) Delete(email string) error {
    tx, err := db.cnx.Begin()
    if err != nil {
        return fmt.Errorf("begin 'delete' transaction failed: %w", err)
    }

    sqlDelete := fmt.Sprintf("DELETE FROM %s WHERE email = ?", tableName)
    stmt, err := tx.Prepare(sqlDelete)
    if err != nil {
        return fmt.Errorf("prepare 'delete' transaction failed: %w", err)
    }
    defer stmt.Close()

    _, err = stmt.Exec(email)
    if err != nil {
        return fmt.Errorf("exec 'delete' transaction failed: %w", err)
    }

    tx.Commit()

    return nil
}
```

## Eliminar el usuario modificado durante el test

Una vez que hemos probado la función `Delete`, podemos usarla como parte de `T.Cleanup` para eliminar el usuario modificado durante el test.

Como sólo estamos interesados en eliminar el usuario, podemos invocar `db.Delete` directamente desde la función de `T.Cleanup`.

Empezamos por el test `TestAdd`:

```go
func TestAdd(t *testing.T) {
    dsn := "file:db4test.db"
    db, email := setupDB(dsn, t)
    u := &gosqlite3.User{
        Email:    gofakeit.Email(),
        Password: gofakeit.Password(true, true, true, true, false, 15),
    }
    if err := db.Add(u); err != nil {
        t.Errorf("failed to insert user: %s", err.Error())
    }
    t.Logf("(add) added email: %s", u.Email)

    t.Cleanup(func() {
        db.Delete(email) // Delete user created by setupDB
        t.Logf("(cleanup) deleted user %s", email)
        db.Delete(u.Email) // Delete user created by TestAdd
        t.Logf("(cleanup) deleted user %s", u.Email)
    })
}
```

Ejecutando los tests en modo *verbose* observamos cómo se eliminan los usuarios creados:

```go
$ go test ./... -v 
=== RUN   TestConnect
--- PASS: TestConnect (0.04s)
=== RUN   TestAdd
    gosqlite3_test.go:73: (setupDB) test email: mozellemoore@balistreri.net
    gosqlite3_test.go:41: (add) added email: jerryschmidt@schulist.org
    gosqlite3_test.go:45: (cleanup) deleted user mozellemoore@balistreri.net
    gosqlite3_test.go:47: (cleanup) deleted user jerryschmidt@schulist.org
--- PASS: TestAdd (0.02s)
=== RUN   TestDelete
    gosqlite3_test.go:73: (setupDB) test email: lelahjacobi@zemlak.io
    gosqlite3_test.go:54: (delete): test email: lelahjacobi@zemlak.io
--- PASS: TestDelete (0.01s)
=== RUN   Test_setupDB
    gosqlite3_test.go:73: (setupDB) test email: narcisoolson@schneider.info
--- PASS: Test_setupDB (0.00s)
PASS
ok      github.com/xaviatwork/gosqlite3/gosqlite3       0.144s
```

Repetimos el proceso para la función `Test_setupDB`:

```go
func Test_setupDB(t *testing.T) {
    dsn := "file:db4test.db"
    db, email := setupDB(dsn, t)
    if db == nil || email == "" {
        t.Errorf("db setup failed with no error")
    }
    t.Cleanup(func() {
        db.Delete(email) // Delete user created by Test_setupDB
        t.Logf("(cleanup) deleted user %s", email)
    })
}
```

Y ahora vemos que, tras cada test, se eliminan los usuarios generados, por lo que la base de datos se deja como estaba antes del test:

```console
$ go test ./... -v 
=== RUN   TestConnect
--- PASS: TestConnect (0.04s)
=== RUN   TestAdd
    gosqlite3_test.go:73: (setupDB) test email: eugenemueller@mckenzie.io
    gosqlite3_test.go:41: (add) added email: ezratorphy@greenholt.name
    gosqlite3_test.go:45: (cleanup) deleted user eugenemueller@mckenzie.io
    gosqlite3_test.go:47: (cleanup) deleted user ezratorphy@greenholt.name
--- PASS: TestAdd (0.02s)
=== RUN   TestDelete
    gosqlite3_test.go:73: (setupDB) test email: austynzulauf@runolfsdottir.info
    gosqlite3_test.go:54: (delete): test email: austynzulauf@runolfsdottir.info
--- PASS: TestDelete (0.03s)
=== RUN   Test_setupDB
    gosqlite3_test.go:73: (setupDB) test email: anahimohr@fay.name
    gosqlite3_test.go:85: (cleanup) deleted user anahimohr@fay.name
--- PASS: Test_setupDB (0.03s)
PASS
ok      github.com/xaviatwork/gosqlite3/gosqlite3       0.203s
```

## Obtener un usuario de la base de datos (a partir de su *email*)

La función de `setupDB` genera un usuario.
Consultaremos si este usuario se encuentra en la base de datos. Si el *email* coincide con un usuario registrado, lo devolvemos. En caso contrario, devolvemos el error que se ha producido.

Empezamos con el test:

```go
func TestGet(t *testing.T) {
    dsn := "file:db4test.db"
    db, email := setupDB(dsn, t)
    t.Logf("(get) email: %s", email)
    u, err := db.Get(email)
    if err != nil {
        t.Errorf("failed to get user %s, %s", email, err.Error())
    }
    if u.Email != email {
        t.Errorf("error retrieving user; got %s but wanted %s", u.Email, email)
    }
}
```

Conseguimos que compile con:

```go
func (db *Database) Get(email string) error {
    return &User{}, nil
}
```

Refactorizamos:

```go
func (db *Database) Get(email string) (*User, error) {
    sqlGet := fmt.Sprintf("SELECT * FROM %s WHERE email = ?", tableName)
    stmt, err := db.cnx.Prepare(sqlGet)
    if err != nil {
        return &User{}, err
    }

    u := &User{}
    err = stmt.QueryRow(email).Scan(&u.Email, &u.Password)
    if err != nil {
        return &User{}, fmt.Errorf("exec 'delete' transaction failed: %w", err)
    }

    return u, nil
}
```

Finalmente, eliminamos el usuario en el *cleanup*:

```go
func TestGet(t *testing.T) {
    dsn := "file:db4test.db"
    db, email := setupDB(dsn, t)
    t.Logf("(get) email: %s", email)
    u, err := db.Get(email)
    if err != nil {
        t.Errorf("failed to get user %s, %s", email, err.Error())
    }
    if u.Email != email {
        t.Errorf("error retrieving user; got %s but wanted %s", u.Email, email)
    }

    t.Cleanup(func() {
        db.Delete(email)
        t.Logf("(cleanup) deleted user %s", u.Email)
    })
}
```

## Actualizar un usuario

A la hora de actualizar un usuario, en nuestro caso sólo podemos modificar la contraseña.
En cualquier caso, la acción de actualizar un registro requiere pasar todos los campos, independientemente de si se han modificado o no.

Empezamos creando el test:

```go
func TestUpdate(t *testing.T) {
    dsn := "file:db4test.db"
    db, email := setupDB(dsn, t)
    t.Logf("(update) email: %s", email)
    u, err := db.Get(email)
    if err != nil {
        t.Errorf("failed to get user %s, %s", email, err.Error())
    }

    u.Password = "updated_p@55w0rD"
    if err := db.Update(u); err != nil {
        t.Errorf("error updating user %s: %s", u.Email, err.Error())
    }

    t.Cleanup(func() {
        db.Delete(u.Email)
        t.Logf("(cleanup) deleted user %s", u.Email)
    })
}
```

Obtenemos el valor de un usuario existente en la bbdd, actualizamos el *password* y lo actualizamos. Si no se produce un error, consideramos que el usuario se ha actualizado correctamente.

Al tratarse de una operación de escritura en la base de datos, usamos una *transacción*:

```go
func (db *Database) Update(u *User) error {
    tx, err := db.cnx.Begin()
    if err != nil {
        return fmt.Errorf("begin 'update' transaction failed: %w", err)
    }

    sqlUpdate := fmt.Sprintf("UPDATE %s SET email = ?, password = ? WHERE email = ?", tableName)
    stmt, err := tx.Prepare(sqlUpdate)
    if err != nil {
        return fmt.Errorf("prepare 'update' transaction failed: %w", err)
    }
    defer stmt.Close()

    _, err = stmt.Exec(u.Email, u.Password, u.Email)
    if err != nil {
        return fmt.Errorf("exec 'update' transaction failed: %w", err)
    }

    tx.Commit()

    return nil
}
```