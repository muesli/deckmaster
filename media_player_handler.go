package main

func handleMediaPlayerStatusChanged() {
	deck.updateWidgets()
}

func handleMediaPlayerActivePlayerChanged() {
	deck.updateWidgets()
}
