package main

type Client interface {
	Open() error
	Close() error
}
