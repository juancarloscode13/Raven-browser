window.musicAPI = {
  async request(path, options = {}) {
    const response = await fetch(path, options);
    const body = await response.json().catch(() => ({}));
    if (!response.ok) throw new Error(body.error || `Error del servidor (${response.status})`);
    return body;
  },
  songs() { return this.request('/api/songs'); },
  scan() { return this.request('/api/scan', { method: 'POST' }); },
  audioURL(id) { return `/api/songs/${encodeURIComponent(id)}/audio`; },
  json(path, method, value) { return this.request(path, { method, headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(value) }); },
  playlists() { return this.request('/api/playlists'); },
  playlistSongs(id) { return this.request(`/api/playlists/${encodeURIComponent(id)}/songs`); },
  reorderPlaylist(id, songIDs) { return this.json(`/api/playlists/${encodeURIComponent(id)}/songs`, 'PUT', { song_ids: songIDs }); },
  createPlaylist(name) { return this.json('/api/playlists', 'POST', { name }); },
  renamePlaylist(id, name) { return this.json(`/api/playlists/${encodeURIComponent(id)}`, 'PATCH', { name }); },
  deletePlaylist(id) { return this.request(`/api/playlists/${encodeURIComponent(id)}`, { method: 'DELETE' }); },
  setPlaylistSong(playlistID, songID, present) {
    const method = present ? 'PUT' : 'DELETE';
    return this.request(`/api/playlists/${encodeURIComponent(playlistID)}/songs/${encodeURIComponent(songID)}`, { method });
  },
  tags() { return this.request('/api/tags'); },
  createTag(name) { return this.json('/api/tags', 'POST', { name }); },
  deleteTag(id) { return this.request(`/api/tags/${encodeURIComponent(id)}`, { method: 'DELETE' }); },
  setSongTag(songID, tagID, present) {
    const method = present ? 'PUT' : 'DELETE';
    return this.request(`/api/songs/${encodeURIComponent(songID)}/tags/${encodeURIComponent(tagID)}`, { method });
  },
  organization(songID) { return this.request(`/api/songs/${encodeURIComponent(songID)}/organization`); }
};
