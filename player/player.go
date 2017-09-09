package player

import (
	"fmt"
	"io/ioutil"
	"path"
	"flag"
	"github.com/Sirupsen/logrus"

	"github.com/veandco/go-sdl2/mix"
)

var songpath = flag.String("path", "songs", "songs path")

type Song struct {
	Name     string
	Filename string
}

func NewSong(filename string) Song {
	logrus.Infof("New song %s", filename)
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

	mix.HookMusicFinished(func() {
		p.playing = 0
		p.PlayNext()
	})

	files, err := ioutil.ReadDir(*songpath)
	if err != nil {
		logrus.Error("Failed to read dir %s: %s", *songpath, err.Error())
		return err
	}

	for _, file := range files {
		p.playlist = append(p.playlist, NewSong(path.Join(*songpath,file.Name())))
	}


	return nil
}

func (p *Player) Play(index int) error {
	if len(p.playlist) <= index || index < 0 {
		return fmt.Errorf("Index not found in playlist")
	}

	song := p.playlist[index]
	p.playing = index + 1
	p.SendUpdate()

	err := p.play(song.Filename)
	if err != nil {
		p.playing = 0
		p.SendUpdate()
		logrus.Infof("Failed playback of %s: %s", song.Filename, err.Error())
	}
	return err
}

func (p *Player) PlayNext() error {
	if mix.PlayingMusic() {
		return nil
	}

	if len(p.queue) == 0 {
		p.SendUpdate()
		return nil
	}

	next := p.queue[0]
	p.queue = p.queue[1:]

	return p.Play(next)
}

func (p *Player) SendUpdate() {
	for _, v := range p.playlistChangeCallbacks {
		v(p.playing, p.queue)
	}
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
			p.SendUpdate()
			return nil
		}
	}

	// Add to the queue
	p.queue = append(p.queue, index)

	// Update the arduino
	p.SendUpdate()

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
