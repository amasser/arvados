package main

import (
	"errors"
	"log"
	"time"
)

/*
	Keepstore initiates trash worker channel goroutine.
	The channel will process trash list.
		For each (next) trash request:
      Delete the block indicated by the trash request Locator
		Repeat
*/

func RunTrashWorker(trashq *WorkQueue) {
	for item := range trashq.NextItem {
		trashRequest := item.(TrashRequest)
		TrashItem(trashRequest)
		trashq.DoneItem <- struct{}{}
	}
}

// TrashItem deletes the indicated block from every writable volume.
func TrashItem(trashRequest TrashRequest) {
	reqMtime := time.Unix(trashRequest.BlockMtime, 0)
	if time.Since(reqMtime) < blob_signature_ttl {
		log.Printf("WARNING: data manager asked to delete a %v old block %v (BlockMtime %d = %v), but my blob_signature_ttl is %v! Skipping.",
			time.Since(reqMtime),
			trashRequest.Locator,
			trashRequest.BlockMtime,
			reqMtime,
			blob_signature_ttl)
		return
	}

	for _, volume := range KeepVM.AllWritable() {
		mtime, err := volume.Mtime(trashRequest.Locator)
		if err != nil {
			log.Printf("%v Delete(%v): %v", volume, trashRequest.Locator, err)
			continue
		}
		if trashRequest.BlockMtime != mtime.Unix() {
			log.Printf("%v Delete(%v): mtime on volume is %v does not match trash list value %v", volume, trashRequest.Locator, mtime.Unix(), trashRequest.BlockMtime)
			continue
		}

		if never_delete {
			err = errors.New("did not delete block because never_delete is true")
		} else {
			err = volume.Delete(trashRequest.Locator)
		}

		if err != nil {
			log.Printf("%v Delete(%v): %v", volume, trashRequest.Locator, err)
		} else {
			log.Printf("%v Delete(%v) OK", volume, trashRequest.Locator)
		}
	}
}
