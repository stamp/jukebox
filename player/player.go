package player

import (
	"fmt"
	"io/ioutil"
	"path"

	"github.com/veandco/go-sdl2/mix"
)

type Song struct {
	Name     string
	Filename string
}

func NewSong(filename string) Song {
	return Song{
		Name:     path.Base(filename),
		Filename: filename,
	}
}

type Player struct {
	playlist []Song

	queue   []int
	playing int

	playlistChangeCallbacks []func(int, []int)
}

func (p *Player) Start() error {
	err := mix.OpenAudio(mix.DEFAULT_FREQUENCY, mix.DEFAULT_FORMAT, mix.DEFAULT_CHANNELS, 4096)
	if err != nil {
		return err
	}

	files, err := ioutil.ReadDir("songs")
	if err != nil {
		return err
	}

	for _, file := range files {
		p.playlist = append(p.playlist, NewSong("songs/"+file.Name()))
	}

	mix.HookMusicFinished(func() {
		p.playing = 0
		p.PlayNext()
	})

	return nil
}

func (p *Player) Play(index int) error {
	if len(p.playlist) <= index || index < 0 {
		return fmt.Errorf("Index not found in playlist")
	}

	song := p.playlist[index]
	p.playing = index + 1

	for _, v := range p.playlistChangeCallbacks {
		v(p.playing, p.queue)
	}

	return p.play(song.Filename)
}

func (p *Player) PlayNext() error {
	if mix.PlayingMusic() {
		return nil
	}

	if len(p.queue) == 0 {
		for _, v := range p.playlistChangeCallbacks {
			v(p.playing, p.queue)
		}
		return nil
	}

	next := p.queue[0]
	p.queue = p.queue[1:]

	return p.Play(next)
}

func (p *Player) Queue(index int) error {
	// Stop the song if you push the button that is playing
	if index == p.playing-1 {
		mix.HaltMusic()
		return nil
	}

	for k, v := range p.queue {
		if v == index {
			p.queue = append(p.queue[:k], p.queue[k+1:]...)
			for _, v := range p.playlistChangeCallbacks {
				v(p.playing, p.queue)
			}
			return nil
		}
	}

	// Add to the queue
	p.queue = append(p.queue, index)

	// Update the arduino
	for _, v := range p.playlistChangeCallbacks {
		v(p.playing, p.queue)
	}

	// Start playback if it isnt already running
	return p.PlayNext()
}

func (p *Player) OnPlaylistChange(cb func(int, []int)) {
	p.playlistChangeCallbacks = append(p.playlistChangeCallbacks, cb)
}

func (p *Player) play(fileName string) error {
	music, err := mix.LoadMUS(fileName)
	if err != nil {
		return err
	}

	return music.Play(0)
}
