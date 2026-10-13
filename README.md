# Reproductor musical local

Aplicación local para catalogar y reproducir archivos de música, crear listas y asignar tags. Los archivos de audio permanecen en su carpeta original; la base de datos solo guarda el catálogo y la organización.

## Requisitos

- Go 1.26 o posterior.
- No hace falta instalar SQLite por separado. El driver `modernc.org/sqlite` está escrito en Go y no requiere CGO.
- Un navegador con soporte para el elemento HTML `<audio>`.

## Compilar y ejecutar

Desde la raíz del proyecto:

```sh
go build -o local-music .
./local-music
```

En Windows, se puede compilar con `go build -o local-music.exe .` y ejecutar `./local-music.exe`.

Por defecto, la aplicación crea `music/` y `data/` en el directorio de trabajo actual, escanea `music/` al arrancar y sirve la interfaz en <http://127.0.0.1:8080>. El escaneo manual está disponible desde el botón **Escanear biblioteca**. Los errores del escaneo inicial se registran en la consola; el servidor sigue disponible para mostrar el catálogo y permitir corregir la carpeta.

### Configuración

Las variables son opcionales:

| Variable | Valor predeterminado | Uso |
| --- | --- | --- |
| `MUSIC_DIR` | `music` | Carpeta que se escanea. Puede ser una ruta absoluta o relativa. |
| `MUSIC_DB` | `data/music.db` | Archivo SQLite local. |
| `MUSIC_ADDR` | `127.0.0.1:8080` | Dirección HTTP de escucha. |

Ejemplo en Linux/macOS para seleccionar una biblioteca existente:

```sh
MUSIC_DIR="$HOME/Música" ./local-music
```

En PowerShell:

```powershell
$env:MUSIC_DIR = 'D:\Mi música'
./local-music.exe
```

La dirección predeterminada solo acepta conexiones de este equipo. Si se cambia `MUSIC_ADDR`, hay que elegir conscientemente la interfaz de red deseada.

## Uso

- La biblioteca se actualiza al iniciar y desde **Escanear biblioteca**.
- La búsqueda filtra en tiempo real por título, artista y tags. También se aplica a la lista que se esté viendo.
- Pulsa **Reproducir** para iniciar una canción. La cola se forma con las canciones visibles; los controles permiten pausar, avanzar, retroceder, ajustar el volumen y mover el progreso.
- Usa **Listas** para crear, renombrar, eliminar y abrir listas. En una lista, las flechas cambian su orden. El menú contextual de cada canción permite añadirla o quitarla de listas.
- Usa **Tags** para crear y borrar tags. El menú contextual permite asignarlos o quitarlos de cada canción.
- Eliminar una lista no borra canciones. Eliminar un tag sí elimina sus asignaciones.

## Nombres y formatos

Se consideran los archivos `.mp3`, `.wav`, `.flac`, `.ogg`, `.opus`, `.m4a` y `.aac`, sin inspeccionar sus metadatos internos. Se intenta separar título y artista por el último guion bajo del nombre sin extensión: `titulo_artista.mp3`. Si falta esa separación, el nombre completo queda como título y el artista se muestra como `Desconocido`.

El escáner solo cataloga archivos regulares y no sigue enlaces simbólicos. Guarda rutas relativas a la carpeta elegida e identifica cada registro por su ruta. Al faltar un archivo conserva la canción, sus listas y sus tags, y la marca como no disponible; si vuelve a aparecer en la misma ruta, recupera disponibilidad.

La reproducción usa el audio original y las peticiones HTTP Range; no hay transcodificación. Que un formato se catalogue no garantiza que el navegador pueda decodificarlo. La compatibilidad depende del navegador, el sistema operativo y, en algunos casos, de los códecs instalados.

## Datos y seguridad

`data/music.db` es la fuente de verdad de listas, tags y catálogo. El esquema se crea y actualiza mediante migraciones versionadas. La aplicación no sobrescribe voluntariamente una base de datos que no pueda abrir ni elimina registros de organización durante un escaneo.

El endpoint de audio recibe el identificador del catálogo, no una ruta de disco. La ruta relativa se abre bajo la raíz musical y se rechazan escapes fuera de ella. Los recursos HTML, CSS y JavaScript están incluidos en el ejecutable.

## Pruebas y comprobaciones

```sh
go test ./...
go build ./...
```

Las pruebas cubren migraciones, restricciones SQLite, el escáner, persistencia y disponibilidad de canciones, relaciones de listas y tags, validación de la API y peticiones HTTP Range. No hay pruebas automatizadas de interacción con un navegador ni pruebas de decodificación con muestras reales de cada códec. Los formatos enumerados están comprobados como extensiones reconocidas por el escáner; la reproducción efectiva depende del navegador.

## Limitaciones conocidas

- No lee metadatos de audio ni calcula duración, carátulas o formas de onda.
- No transcodifica ni garantiza soporte de códecs por parte del navegador.
- Una canción que se mueve a otra ruta recibe un registro nuevo; no se reasocia por título o artista.
- La búsqueda se ejecuta en el cliente sobre el catálogo cargado en memoria.
- No hay usuarios, acceso remoto previsto ni sincronización entre dispositivos.
