package main

import (
	"errors"
	"image"
	"net/http"
	"sync"
	"time"
)

type ImageDownloader struct {
	ttl time.Duration

	cache map[string]CachedImage
	mutex *sync.RWMutex
}

type CachedImage struct {
	Expires time.Time
	Image   image.Image
}

func NewImageDownloader(ttl time.Duration) *ImageDownloader {
	return &ImageDownloader{
		ttl: ttl,

		cache: make(map[string]CachedImage),
		mutex: &sync.RWMutex{},
	}
}

func (d *ImageDownloader) Watch() {
	go func() {
		time.Sleep(1 * time.Minute)

		now := time.Now()
		d.mutex.Lock()
		for url, c := range d.cache {
			if now.After(c.Expires) {
				verbosef("removing expired image %s from cache", url)
				delete(d.cache, url)
			}
		}
		d.mutex.Unlock()
	}()
}

func (d *ImageDownloader) Download(url string) (image.Image, error) {
	cachedImage := d.fromCache(url)
	if cachedImage != nil {
		return cachedImage, nil
	}

	response, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return nil, errors.New("unable to download image from URL")
	}
	img, _, err := image.Decode(response.Body)
	if err != nil {
		return nil, err
	}

	d.mutex.Lock()
	d.cache[url] = CachedImage{
		Expires: time.Now().Add(d.ttl),
		Image:   img,
	}
	d.mutex.Unlock()

	verbosef("saving image %s in cache", url)

	return img, nil
}

func (d *ImageDownloader) fromCache(url string) image.Image {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	if cache, ok := d.cache[url]; ok {
		if time.Now().Before(cache.Expires) {
			verbosef("returning image %s from cache", url)
			return cache.Image
		}
		verbosef("cache expired for image %s", url)
	}

	return nil
}
