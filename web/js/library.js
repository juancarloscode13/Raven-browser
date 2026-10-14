class MusicLibrary {
  constructor(elements, api, player, onContext) {
    this.elements = elements;
    this.api = api;
    this.player = player;
    this.onContext = onContext;
    this.allSongs = [];
    this.baseSongs = [];
    this.query = '';
    this.navigation = { type: 'all', id: null, name: 'Todas las canciones' };
    this.onReorder = null;
    this.onNavigationChange = null;
    this.navigationRequest = 0;
    this.songsRequest = 0;
    this.loadError = '';
    this.libraryError = '';
    window.addEventListener('raven:trackchange', event => this.markCurrentSong(event.detail.songID));
  }

  async load() {
    const navigationRequest = ++this.navigationRequest;
    const request = ++this.songsRequest;
    this.elements.status.classList.remove('is-error');
    this.elements.status.textContent = 'Cargando biblioteca…';
    this.loadError = '';
    this.libraryError = '';
    try {
      const songs = await this.api.songs();
      if (request !== this.songsRequest) return;
      this.allSongs = songs;
      if (navigationRequest === this.navigationRequest) {
        await this.restoreNavigation(this.navigationRequest);
      } else if (this.navigation.type !== 'playlist') {
        this.restoreLocalNavigation();
        this.display();
      }
    } catch (error) {
      if (request !== this.songsRequest) return;
      this.libraryError = `No se pudo cargar la biblioteca: ${error.message}`;
      this.display();
    }
  }

  async refreshSongs() {
    const request = ++this.songsRequest;
    try {
      const songs = await this.api.songs();
      if (request !== this.songsRequest) return;
      this.allSongs = songs;
      this.libraryError = '';
      if (this.navigation.type === 'playlist') {
        await this.selectPlaylist(this.navigation, { notify: false });
      } else {
        this.restoreLocalNavigation();
        this.display();
      }
    } catch (error) {
      if (request !== this.songsRequest) return;
      this.libraryError = `No se pudo actualizar la biblioteca: ${error.message}`;
      this.display();
    }
  }

  async restoreNavigation(request = this.navigationRequest) {
    if (this.navigation.type === 'playlist') {
      try {
        const songs = await this.api.playlistSongs(this.navigation.id);
        if (request !== this.navigationRequest) return;
        this.baseSongs = songs;
      } catch (error) {
        if (request !== this.navigationRequest) return;
        this.loadError = `No se pudo cargar la lista: ${error.message}`;
      }
    } else {
      this.restoreLocalNavigation();
    }
    this.display();
  }

  restoreLocalNavigation() {
    this.onReorder = null;
    if (this.navigation.type === 'tag') {
      const target = this.navigation.name.toLocaleLowerCase();
      this.baseSongs = this.allSongs.filter(song => (song.tags || []).some(tag => tag.toLocaleLowerCase() === target));
    } else {
      this.baseSongs = this.allSongs;
    }
  }

  selectAll() {
    this.navigationRequest++;
    this.navigation = { type: 'all', id: null, name: 'Todas las canciones' };
    this.loadError = '';
    this.restoreLocalNavigation();
    this.navigationChanged();
    this.display();
  }

  async selectPlaylist(playlist, options = {}) {
    const request = ++this.navigationRequest;
    const selection = { type: 'playlist', id: playlist.id, name: playlist.name };
    this.navigation = selection;
    this.onReorder = null;
    this.loadError = '';
    if (options.notify !== false) this.navigationChanged();
    this.elements.status.classList.remove('is-error');
    this.elements.status.textContent = `Cargando lista «${playlist.name}»…`;
    try {
      const songs = await this.api.playlistSongs(playlist.id);
      if (request !== this.navigationRequest) return;
      this.baseSongs = songs;
      this.onReorder = async ordered => {
        await this.api.reorderPlaylist(playlist.id, ordered.map(song => song.id));
        if (this.navigation.type === 'playlist' && this.navigation.id === playlist.id) {
          await this.selectPlaylist({ id: playlist.id, name: this.navigation.name }, { notify: false });
        }
      };
      this.display();
    } catch (error) {
      if (request !== this.navigationRequest) return;
      this.loadError = `No se pudo cargar la lista: ${error.message}`;
      this.baseSongs = [];
      this.display();
    }
  }

  selectTag(tag) {
    this.navigationRequest++;
    this.navigation = { type: 'tag', id: tag.id, name: tag.name };
    this.loadError = '';
    this.restoreLocalNavigation();
    this.navigationChanged();
    this.display();
  }

  renameNavigation(id, name) {
    if (this.navigation.type !== 'playlist' || this.navigation.id !== id) return;
    this.navigation.name = name;
    this.navigationChanged();
    this.display();
  }

  async removeNavigation(id, type) {
    if (this.navigation.type === type && this.navigation.id === id) this.selectAll();
  }

  setQuery(query) {
    this.query = query.trim().toLocaleLowerCase();
    this.display();
  }

  navigationChanged() {
    if (typeof this.onNavigationChange === 'function') this.onNavigationChange(this.navigation);
  }

  showError(error) {
    this.libraryError = `No se pudo cargar la biblioteca: ${error.message}`;
    this.display();
  }

  display() {
    let songs = this.baseSongs;
    if (this.query) {
      songs = this.allSongs.filter(song =>
        [song.title, song.artist, ...(song.tags || [])].some(value => value.toLocaleLowerCase().includes(this.query))
      );
    }

    if (this.elements.title) {
      this.elements.title.textContent = this.query ? 'Resultados de búsqueda' : this.navigation.name;
    }
    const hasDisplayError = Boolean(this.libraryError || (this.loadError && !this.query));
    this.elements.status.classList.toggle('is-error', hasDisplayError);
    if (hasDisplayError) {
      this.elements.status.textContent = this.libraryError || this.loadError;
      this.elements.empty.textContent = 'No se pudo mostrar este contenido. Comprueba la conexión con el servidor e inténtalo de nuevo.';
      this.render([]);
      return;
    }

    const countLabel = `${songs.length} ${songs.length === 1 ? 'canción' : 'canciones'}`;
    if (this.query) {
      this.elements.status.textContent = `Búsqueda global · ${songs.length} ${songs.length === 1 ? 'resultado' : 'resultados'} en ${this.allSongs.length} canciones`;
      this.elements.empty.textContent = `No se encontraron canciones para «${this.query}» en toda la biblioteca.`;
      this.render(songs, null);
      return;
    }

    this.elements.status.textContent = this.navigation.type === 'all'
      ? countLabel
      : `${this.navigation.type === 'tag' ? '#' : ''}${this.navigation.name} · ${countLabel}`;
    this.elements.empty.textContent = this.emptyMessage(songs.length);
    this.render(songs, this.navigation.type === 'playlist' ? this.onReorder : null);
  }

  emptyMessage(count) {
    if (count) return '';
    if (this.navigation.type === 'playlist') return 'Esta lista todavía no contiene canciones. Añade canciones desde el menú de organización.';
    if (this.navigation.type === 'tag') return `Todavía no hay canciones con el tag «${this.navigation.name}».`;
    return 'Aún no hay canciones. Añade archivos a la carpeta musical y escanea la biblioteca.';
  }

  markCurrentSong(songID) {
    this.elements.list.querySelectorAll('[data-song-id]').forEach(row => {
      const isCurrent = row.dataset.songId === String(songID);
      row.classList.toggle('is-current', isCurrent);
      if (isCurrent) row.setAttribute('aria-current', 'true');
      else row.removeAttribute('aria-current');
    });
  }

  render(songs, onReorder = null) {
    const body = this.elements.list;
    const currentSong = this.player.queue?.[this.player.index];
    body.replaceChildren();
    songs.forEach((song, index) => {
      const row = document.createElement('tr');
      row.className = `song-row${song.available ? ' is-playable' : ''}`;
      row.dataset.songId = String(song.id);
      if (currentSong && String(currentSong.id) === String(song.id)) {
        row.classList.add('is-current');
        row.setAttribute('aria-current', 'true');
      }
      if (song.available) {
        row.tabIndex = 0;
        row.setAttribute('aria-label', `Reproducir ${song.title} de ${song.artist}`);
        row.setAttribute('aria-keyshortcuts', 'Enter Space');
      }
      const title = document.createElement('td'); title.textContent = song.title;
      if (!song.available) {
        const availability = document.createElement('span');
        availability.className = 'availability-status';
        availability.textContent = 'No disponible';
        title.append(availability);
      }
      const artist = document.createElement('td'); artist.textContent = song.artist;
      const format = document.createElement('td'); format.textContent = (song.format || '').toUpperCase(); format.className = 'format';
      const action = document.createElement('td');
      const controls = document.createElement('div'); controls.className = 'row-actions';
      const organize = document.createElement('button'); organize.type = 'button'; organize.className = 'small-action';
      organize.innerHTML = '<svg viewBox="0 0 24 24" class="icon" aria-hidden="true" focusable="false"><circle cx="5" cy="12" r="1" /><circle cx="12" cy="12" r="1" /><circle cx="19" cy="12" r="1" /></svg>';
      organize.setAttribute('aria-label', `Gestionar tags y listas de ${song.title}`);
      organize.addEventListener('click', event => this.onContext(song, { clientX: event.clientX, clientY: event.clientY }));
      if (onReorder) {
        const move = (isUp, delta) => {
          const control = document.createElement('button'); control.type = 'button'; control.className = 'small-action order-action';
          control.innerHTML = isUp
            ? '<svg viewBox="0 0 24 24" class="icon" aria-hidden="true" focusable="false"><path d="m6 14 6-6 6 6" /></svg>'
            : '<svg viewBox="0 0 24 24" class="icon" aria-hidden="true" focusable="false"><path d="m6 10 6 6 6-6" /></svg>';
          control.setAttribute('aria-label', `${isUp ? 'Subir' : 'Bajar'} ${song.title}`);
          control.disabled = !songs[index + delta];
          control.addEventListener('click', async () => {
            const ordered = [...songs];
            [ordered[index], ordered[index + delta]] = [ordered[index + delta], ordered[index]];
            try { await onReorder(ordered); }
            catch (error) {
              this.elements.status.classList.add('is-error');
              this.elements.status.textContent = `No se pudo cambiar el orden: ${error.message}`;
            }
          });
          return control;
        };
        controls.append(move(true, -1), move(false, 1));
      }
      controls.append(organize);
      action.append(controls);
      row.addEventListener('click', event => {
        if (!song.available || (event.target instanceof Element && event.target.closest('button, a, input, select, textarea, [contenteditable="true"]'))) return;
        this.player.setQueue(songs, song.id);
      });
      row.addEventListener('keydown', event => {
        if (!song.available || event.target !== row || event.repeat || (event.key !== 'Enter' && event.key !== ' ')) return;
        if (event.key === ' ') event.preventDefault();
        this.player.setQueue(songs, song.id);
      });
      row.addEventListener('contextmenu', event => { event.preventDefault(); this.onContext(song, event); });
      row.append(title, artist, format, action); body.append(row);
    });
    this.elements.empty.hidden = songs.length !== 0;
    this.elements.table.hidden = songs.length === 0;
  }
}

window.MusicLibrary = MusicLibrary;
