class MusicPlayer {
  constructor(elements, api) {
    this.audio = elements.audio;
    this.elements = elements;
    this.api = api;
    this.queue = [];
    this.index = -1;
    this.audio.volume = Number(elements.volume.value);
    elements.playPause.disabled = true;
    elements.previous.disabled = true;
    elements.next.disabled = true;
    elements.playPause.addEventListener('click', () => this.toggle());
    elements.previous.addEventListener('click', () => this.previous());
    elements.next.addEventListener('click', () => this.next());
    elements.volume.addEventListener('input', () => { this.audio.volume = Number(elements.volume.value); });
    elements.progress.addEventListener('input', () => {
      if (Number.isFinite(this.audio.duration)) this.audio.currentTime = this.audio.duration * Number(elements.progress.value) / 1000;
    });
    this.audio.addEventListener('timeupdate', () => this.updateTime());
    this.audio.addEventListener('loadedmetadata', () => this.updateTime());
    this.audio.addEventListener('play', () => {
      elements.playPause.setAttribute('aria-label', 'Pausar');
      elements.playPause.setAttribute('aria-pressed', 'true');
    });
    this.audio.addEventListener('pause', () => {
      elements.playPause.setAttribute('aria-label', 'Reproducir');
      elements.playPause.setAttribute('aria-pressed', 'false');
    });
    this.audio.addEventListener('ended', () => this.next());
    this.audio.addEventListener('error', () => { if (this.index >= 0) this.message('No se pudo reproducir este archivo. Puede faltar o no ser compatible con el navegador.'); });
  }

  setQueue(songs, selectedID) {
    this.queue = songs.filter(song => song.available);
    this.index = this.queue.findIndex(song => song.id === selectedID);
    this.renderQueue();
    return this.playIndex(this.index);
  }

  async playIndex(index) {
    if (index < 0 || index >= this.queue.length) return;
    this.index = index;
    const song = this.queue[index];
    this.audio.src = this.api.audioURL(song.id);
    this.elements.nowTitle.textContent = song.title;
    this.elements.nowArtist.textContent = song.artist;
    this.elements.progress.value = '0';
    this.elements.currentTime.textContent = '0:00';
    this.elements.duration.textContent = '0:00';
    this.renderQueue();
    window.dispatchEvent(new CustomEvent('raven:trackchange', { detail: { songID: song.id } }));
    this.message(`Reproduciendo ${song.title} — ${song.artist}`);
    try { await this.audio.play(); }
    catch { this.message('El navegador no pudo iniciar la reproducción. Comprueba el formato del archivo.'); }
  }

  async toggle() {
    if (!this.audio.src) return;
    if (this.audio.paused) {
      try { await this.audio.play(); } catch { this.message('No se pudo iniciar la reproducción.'); }
    } else this.audio.pause();
  }

  previous() {
    if (this.index < 0) return;
    if (this.audio.currentTime > 3 || this.index === 0) this.audio.currentTime = 0;
    else this.playIndex(this.index - 1);
  }

  next() { if (this.index >= 0 && this.index < this.queue.length - 1) this.playIndex(this.index + 1); }

  updateTime() {
    const duration = this.audio.duration;
    this.elements.currentTime.textContent = this.format(this.audio.currentTime);
    this.elements.duration.textContent = this.format(duration);
    this.elements.progress.value = Number.isFinite(duration) && duration > 0 ? String(Math.round(this.audio.currentTime / duration * 1000)) : '0';
  }

  format(seconds) {
    if (!Number.isFinite(seconds)) return '0:00';
    const value = Math.floor(seconds);
    return `${Math.floor(value / 60)}:${String(value % 60).padStart(2, '0')}`;
  }

  renderQueue() {
    const list = this.elements.queueList;
    this.elements.playPause.disabled = this.queue.length === 0;
    this.elements.previous.disabled = this.index < 0;
    this.elements.next.disabled = this.index < 0 || this.index >= this.queue.length - 1;
    list.replaceChildren();
    if (!this.queue.length) {
      const empty = document.createElement('li'); empty.className = 'empty'; empty.textContent = 'La cola está vacía.'; list.append(empty); return;
    }
    this.queue.forEach((song, index) => {
      const item = document.createElement('li');
      const button = document.createElement('button'); button.type = 'button'; button.className = 'queue-item';
      button.textContent = `${song.title} · ${song.artist}`;
      if (index === this.index) button.setAttribute('aria-current', 'true');
      button.addEventListener('click', () => this.playIndex(index));
      item.append(button); list.append(item);
    });
  }

  message(text) { this.elements.status.textContent = text; }
}

window.MusicPlayer = MusicPlayer;
