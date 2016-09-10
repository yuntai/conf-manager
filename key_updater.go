package main

import "time"

type UpdateJob struct {
	r *Repo
}

var updaterQueue = make(chan UpdateJob, 30)

type GitEvent struct {
}

type GitPoller struct {
	r             *Repo
	currentCommit string
	updateCh      chan GitEvent
	interval      time.Duration
	shutdownCh    chan interface{}
}

func (p *GitPoller) Poll() {
	//pollTicker := time.NewTicker(time.Millisecond * time.Duration(config.updateInterval))
	/*
		for {
			select {
			case <-pollTicker.C:
				tip := p.r.getCurrentTip()
				if tip > currentCommit {
					updateCh <- &GitEvent{repo: p.repo, commit: tip}
					currentCommit = tip
				}
			case <-shutdownCh:
				close(updateCh)
				return
			}
		}
	*/
}
