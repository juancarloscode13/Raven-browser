class TagManager {
  constructor(api, list, library) {
    this.api = api;
    this.list = list;
    this.library = library;
    this.items = [];
    this.loaded = false;
  }

  async load() {
    try {
      this.items = await this.api.tags();
      this.loaded = true;
      this.list.classList.remove('is-error');
      this.render();
    } catch (error) {
      this.list.classList.add('is-error');
      this.list.textContent = `No se pudieron cargar los tags: ${error.message}`;
    }
  }

  async ensureLoaded() { if (!this.loaded) await this.load(); }

  async create(name) {
    const tag = await this.api.createTag(name);
    await this.load();
    return tag;
  }

  render() {
    this.list.replaceChildren();
    for (const tag of this.items) {
      const item = document.createElement('li');
      const select = document.createElement('button'); select.type = 'button'; select.className = 'tag-select';
      select.textContent = `# ${tag.name}`;
      if (this.library.navigation.type === 'tag' && this.library.navigation.id === tag.id) select.setAttribute('aria-current', 'page');
      select.addEventListener('click', () => this.library.selectTag(tag));

      const remove = document.createElement('button'); remove.type = 'button';
      remove.innerHTML = '<svg viewBox="0 0 24 24" class="icon" aria-hidden="true" focusable="false"><path d="m6 6 12 12M18 6 6 18" /></svg>';
      remove.className = 'delete-action'; remove.setAttribute('aria-label', `Eliminar tag ${tag.name}`);
      remove.addEventListener('click', async () => {
        if (!window.confirm(`¿Eliminar el tag «${tag.name}» de todas las canciones?`)) return;
        try {
          await this.api.deleteTag(tag.id);
          await this.library.removeNavigation(tag.id, 'tag');
          await this.load();
          await this.library.refreshSongs();
        } catch (error) {
          this.library.elements.status.classList.add('is-error');
          this.library.elements.status.textContent = `No se pudo eliminar el tag: ${error.message}`;
        }
      });
      item.append(select, remove);
      this.list.append(item);
    }
    if (!this.items.length) {
      const empty = document.createElement('li'); empty.className = 'empty'; empty.textContent = 'Crea tags y asígnalos a canciones.';
      this.list.append(empty);
    }
  }
}

class SongOrganizationMenu {
  constructor(api, playlists, tags, elements, onOrganizationChange) {
    this.api = api;
    this.playlists = playlists;
    this.tags = tags;
    this.elements = elements;
    this.onOrganizationChange = onOrganizationChange;
    this.song = null;
    this.openRequest = 0;
    elements.close.addEventListener('click', () => this.close());
    document.addEventListener('pointerdown', event => { if (!elements.menu.hidden && !elements.menu.contains(event.target)) this.close(); });
    document.addEventListener('keydown', event => { if (event.key === 'Escape') this.close(); });
    elements.tagForm.addEventListener('submit', event => { event.preventDefault(); this.createTag(); });
  }

  async open(song, event) {
    this.song = song;
    const request = ++this.openRequest;
    this.elements.title.textContent = song.title;
    this.elements.error.textContent = '';
    this.elements.menu.hidden = false;
    try {
      const [organization] = await Promise.all([
        this.api.organization(song.id), this.playlists.ensureLoaded(), this.tags.ensureLoaded()
      ]);
      if (request !== this.openRequest || !this.song || this.song.id !== song.id) return;
      this.renderChecks(this.elements.playlists, this.playlists.items, organization.playlists, 'playlist');
      this.renderChecks(this.elements.tags, this.tags.items, organization.tags, 'tag');
      const x = Math.min(event.clientX, window.innerWidth - this.elements.menu.offsetWidth - 12);
      const y = Math.min(event.clientY, window.innerHeight - this.elements.menu.offsetHeight - 12);
      this.elements.menu.style.left = `${Math.max(12, x)}px`;
      this.elements.menu.style.top = `${Math.max(12, y)}px`;
      this.elements.close.focus();
    } catch (error) {
      if (request === this.openRequest) this.elements.error.textContent = error.message;
    }
  }

  renderChecks(container, items, assigned, kind) {
    container.replaceChildren();
    const assignedIDs = new Set(assigned.map(item => item.id));
    if (!items.length) {
      const empty = document.createElement('p'); empty.className = 'empty';
      empty.textContent = kind === 'tag' ? 'No hay tags creados.' : 'Crea una lista para empezar.';
      container.append(empty); return;
    }
    for (const item of items) {
      const label = document.createElement('label'); label.className = 'check-option';
      const checkbox = document.createElement('input'); checkbox.type = 'checkbox'; checkbox.checked = assignedIDs.has(item.id);
      checkbox.addEventListener('change', async () => {
        const songID = this.song && this.song.id;
        if (!songID) return;
        checkbox.disabled = true;
        let saved = false;
        try {
          if (kind === 'tag') await this.api.setSongTag(songID, item.id, checkbox.checked);
          else await this.api.setPlaylistSong(item.id, songID, checkbox.checked);
          saved = true;
          if (kind === 'tag') await this.tags.load();
          else await this.playlists.load();
          await this.onOrganizationChange();
          if (this.song && this.song.id === songID) await this.refreshChecks();
        } catch (error) {
          if (!saved) checkbox.checked = !checkbox.checked;
          this.elements.error.textContent = error.message;
        } finally { checkbox.disabled = false; }
      });
      const name = document.createElement('span'); name.textContent = item.name;
      label.append(checkbox, name); container.append(label);
    }
  }

  async createTag() {
    const nameInput = this.elements.tagName;
    if (!this.song) return;
    const songID = this.song.id;
    try {
      const tag = await this.tags.create(nameInput.value);
      nameInput.value = '';
      await this.api.setSongTag(songID, tag.id, true);
      await this.onOrganizationChange();
      if (this.song && this.song.id === songID) await this.refreshChecks();
    } catch (error) { this.elements.error.textContent = error.message; }
  }

  async refreshChecks() {
    if (!this.song) return;
    const organization = await this.api.organization(this.song.id);
    this.renderChecks(this.elements.playlists, this.playlists.items, organization.playlists, 'playlist');
    this.renderChecks(this.elements.tags, this.tags.items, organization.tags, 'tag');
  }

  close() { this.openRequest++; this.elements.menu.hidden = true; this.song = null; }
}

window.TagManager = TagManager;
window.SongOrganizationMenu = SongOrganizationMenu;
