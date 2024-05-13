package main

import (
	"fmt"
	"math/rand/v2"
	"sync"

	"github.com/bazuker/browserbroAPI/client"
)

func main() {
	c, err := client.New("http://localhost:10001", nil)
	if err != nil {
		fmt.Println("failed to create client:", err)
		return
	}
	plugins, err := c.Plugins()
	if err != nil {
		fmt.Println("failed to fetch plugins:", err)
		return
	}
	fmt.Println("available plugins:", plugins)

	runPluginsConcurrently(c, 10, "screenshot", []map[string]any{
		{
			"urls": []string{"https://nowsecure.nl/"},
		},
		{
			"urls": []string{"https://bot.sannysoft.com"},
		},
	})

	runPluginsConcurrently(c, 10, "googlesearch", []map[string]any{
		{
			"query": "golang",
		},
		{
			"query": "javascript",
		},
		{
			"query": "python",
		},
		{
			"query": "java",
		},
		{
			"query": "c++",
		},
		{
			"query": "rust",
		},
	})
}

func runPluginsConcurrently(
	client *client.Client,
	numOfJobs int,
	pluginName string,
	params []map[string]any,
) {
	fmt.Println("running", numOfJobs, "jobs concurrently, plugin:", pluginName)
	success := 0
	var wg sync.WaitGroup
	wg.Add(numOfJobs)
	for i := 0; i < numOfJobs; i++ {
		go func() {
			fmt.Println("running plugin job", i)
			defer wg.Done()
			// pick params at random from the array
			output, err := client.RunPlugin(pluginName, params[rand.IntN(len(params))])
			if err != nil {
				fmt.Println("failed to run plugin:", err)
				return
			}
			fmt.Printf("plugin %d output: %v\n", i, output[pluginName])
			if len(output) > 0 && output[pluginName] != nil {
				success++
			}
		}()
	}

	wg.Wait()
	fmt.Printf("successfully ran %d out of %d jobs\n", success, numOfJobs)
	fmt.Printf("success rate %.2f%%\n", float64(success)/float64(numOfJobs)*100)
}
