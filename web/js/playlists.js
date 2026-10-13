class PlaylistManager {
  constructor(api, list, library) {
    this.api = api;
    this.list = list;
    this.library = library;
    this.items = [];
    this.loaded = false;
  }

  async load() {
    try {
      this.items = await this.api.playlists();
      this.loaded = true;
      this.render();
    } catch (error) {
      this.list.textContent = `No se pudieron cargar las listas: ${error.message}`;
    }
  }

  async ensureLoaded() { if (!this.loaded) await this.load(); }

  async create(name) {
    await this.api.createPlaylist(name);
    await this.load();
  }

  render() {
    this.list.replaceChildren();
    for (const playlist of this.items) {
      const item = document.createElement('li');
      const view = document.createElement('button'); view.type = 'button'; view.className = 'management-name';
      view.textContent = `${playlist.name} (${playlist.song_count})`;
      if (this.library.navigation.type === 'playlist' && this.library.navigation.id === playlist.id) view.setAttribute('aria-current', 'page');
      view.addEventListener('click', () => this.library.selectPlaylist(playlist));

      const rename = document.createElement('button'); rename.type = 'button'; rename.className = 'small-action';
      rename.textContent = 'Editar'; rename.setAttribute('aria-label', `Renombrar ${playlist.name}`);
      rename.addEventListener('click', async () => {
        const name = window.prompt('Nuevo nombre de la lista', playlist.name);
        if (name === null) return;
        try {
          await this.api.renamePlaylist(playlist.id, name);
          this.library.renameNavigation(playlist.id, name.trim());
          await this.load();
        } catch (error) { this.library.elements.status.textContent = `No se pudo renombrar la lista: ${error.message}`; }
      });

      const remove = document.createElement('button'); remove.type = 'button'; remove.className = 'small-action delete-action';
      remove.textContent = '×'; remove.setAttribute('aria-label', `Eliminar ${playlist.name}`);
      remove.addEventListener('click', async () => {
        if (!window.confirm(`¿Eliminar la lista «${playlist.name}»? Las canciones y sus tags se conservarán.`)) return;
        try {
          await this.api.deletePlaylist(playlist.id);
          await this.library.removeNavigation(playlist.id, 'playlist');
          await this.load();
        } catch (error) { this.library.elements.status.textContent = `No se pudo eliminar la lista: ${error.message}`; }
      });
      item.append(view, rename, remove);
      this.list.append(item);
    }
    if (!this.items.length) {
      const empty = document.createElement('li'); empty.className = 'empty'; empty.textContent = 'Todavía no hay listas.';
      this.list.append(empty);
    }
  }
}

window.PlaylistManager = PlaylistManager;
