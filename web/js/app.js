const byID = id => document.getElementById(id);
const player = new MusicPlayer({
  audio: byID('audio'), playPause: byID('play-pause'), previous: byID('previous'), next: byID('next'),
  volume: byID('volume'), progress: byID('progress'), currentTime: byID('current-time'), duration: byID('duration'),
  nowTitle: byID('now-title'), nowArtist: byID('now-artist'), queueList: byID('queue-list'), status: byID('player-status')
}, window.musicAPI);
const library = new MusicLibrary({
  list: byID('song-list'), status: byID('library-status'), empty: byID('empty-state'),
  table: document.querySelector('.table-wrap'), title: byID('library-title')
}, window.musicAPI, player, (song, event) => organizationMenu.open(song, event));
const playlistManager = new PlaylistManager(window.musicAPI, byID('playlist-list'), library);
const tagManager = new TagManager(window.musicAPI, byID('tag-list'), library);
const organizationMenu = new SongOrganizationMenu(window.musicAPI, playlistManager, tagManager, {
  menu: byID('song-menu'), title: byID('menu-song-title'), close: byID('close-song-menu'),
  playlists: byID('menu-playlists'), tags: byID('menu-tags'), tagForm: byID('menu-tag-form'),
  tagName: byID('menu-tag-name'), error: byID('menu-error')
}, () => library.refreshSongs());

library.onNavigationChange = navigation => {
  const all = byID('all-songs');
  const isAll = navigation.type === 'all';
  all.classList.toggle('is-selected', isAll);
  if (isAll) all.setAttribute('aria-current', 'page');
  else all.removeAttribute('aria-current');
  playlistManager.render();
  tagManager.render();
  setSidebarOpen(false, true);
};

function setSidebarOpen(open, restoreFocus = false) {
  const wasOpen = document.body.classList.contains('sidebar-open');
  document.body.classList.toggle('sidebar-open', open);
  const toggle = byID('sidebar-toggle');
  toggle.setAttribute('aria-expanded', String(open));
  toggle.setAttribute('aria-label', open ? 'Cerrar navegación' : 'Abrir navegación');
  if (open) byID('all-songs').focus();
  else if (wasOpen && restoreFocus) toggle.focus();
}

byID('playlist-form').addEventListener('submit', async event => {
  event.preventDefault();
  const input = byID('playlist-name');
  try {
    await playlistManager.create(input.value);
    input.value = '';
    event.currentTarget.classList.remove('is-open');
    document.querySelector('[data-create-target="playlist-name"]').setAttribute('aria-expanded', 'false');
  } catch (error) {
    byID('library-status').classList.add('is-error');
    byID('library-status').textContent = `No se pudo crear la lista: ${error.message}`;
  }
});
byID('tag-form').addEventListener('submit', async event => {
  event.preventDefault();
  const input = byID('tag-name');
  try {
    await tagManager.create(input.value);
    input.value = '';
    event.currentTarget.classList.remove('is-open');
    document.querySelector('[data-create-target="tag-name"]').setAttribute('aria-expanded', 'false');
  } catch (error) {
    byID('library-status').classList.add('is-error');
    byID('library-status').textContent = `No se pudo crear el tag: ${error.message}`;
  }
});
document.querySelectorAll('[data-create-target]').forEach(button => button.addEventListener('click', () => {
  const form = byID(button.dataset.createTarget).closest('form');
  const open = !form.classList.contains('is-open');
  form.classList.toggle('is-open', open);
  button.setAttribute('aria-expanded', String(open));
  if (open) byID(button.dataset.createTarget).focus();
}));

byID('all-songs').addEventListener('click', () => library.selectAll());
byID('search-input').addEventListener('input', event => {
  library.setQuery(event.currentTarget.value);
  byID('clear-search').hidden = event.currentTarget.value.length === 0;
});
byID('clear-search').addEventListener('click', () => {
  byID('search-input').value = '';
  byID('clear-search').hidden = true;
  library.setQuery('');
  byID('search-input').focus();
});
byID('search-input').addEventListener('keydown', event => {
  if (event.key === 'Escape' && event.currentTarget.value) {
    event.currentTarget.value = '';
    byID('clear-search').hidden = true;
    library.setQuery('');
  }
});

byID('scan-button').addEventListener('click', async event => {
  const button = event.currentTarget;
  button.disabled = true;
  byID('library-status').classList.remove('is-error');
  byID('library-status').textContent = 'Escaneando…';
  try {
    const result = await window.musicAPI.scan();
    await library.load();
    if (!library.query && !library.libraryError) byID('library-status').textContent = `Escaneo terminado: ${result.songs} canciones`;
  } catch (error) {
    byID('library-status').classList.add('is-error');
    byID('library-status').textContent = `Error al escanear: ${error.message}`;
  } finally { button.disabled = false; }
});

byID('queue-toggle').addEventListener('click', () => {
  const panel = byID('queue-panel');
  const open = panel.hidden;
  panel.hidden = !open;
  byID('queue-toggle').setAttribute('aria-expanded', String(open));
  if (open) byID('close-queue').focus();
});
byID('close-queue').addEventListener('click', () => {
  byID('queue-panel').hidden = true;
  byID('queue-toggle').setAttribute('aria-expanded', 'false');
  byID('queue-toggle').focus();
});
byID('sidebar-toggle').addEventListener('click', () => {
  const open = !document.body.classList.contains('sidebar-open');
  setSidebarOpen(open, !open);
});
byID('sidebar-close').addEventListener('click', () => setSidebarOpen(false, true));
document.addEventListener('pointerdown', event => {
  if (document.body.classList.contains('sidebar-open') &&
      !byID('sidebar').contains(event.target) && !byID('sidebar-toggle').contains(event.target)) setSidebarOpen(false);
});
document.addEventListener('keydown', event => {
  if (event.key === 'Escape') {
    setSidebarOpen(false, true);
    if (!byID('queue-panel').hidden) {
      byID('queue-panel').hidden = true;
      byID('queue-toggle').setAttribute('aria-expanded', 'false');
      byID('queue-toggle').focus();
    }
  }
});

window.musicAPI.request('/api/health')
  .then(() => Promise.all([library.load(), playlistManager.load(), tagManager.load()]))
  .catch(error => library.showError(error));
