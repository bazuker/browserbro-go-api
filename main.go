package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/bazuker/browserbro-go-api/client"
)

func main() {
	googleSearch := flag.String("googlesearch", "", "run the googlesearch plugin with the given query")
	screenshot := flag.String("screenshot", "", "run the screenshot plugin against the given URL")
	test := flag.Bool("test", false, "run a concurrent load test against the screenshot and googlesearch plugins")
	flag.Parse()

	if *googleSearch == "" && *screenshot == "" && !*test {
		flag.Usage()
		os.Exit(2)
	}

	c, err := client.New("http://localhost:10001", nil)
	if err != nil {
		fmt.Println("failed to create client:", err)
		os.Exit(1)
	}

	if *googleSearch != "" {
		if _, err := runPlugin(c, "googlesearch", map[string]any{
			"query": *googleSearch,
		}); err != nil {
			fmt.Println("failed to run googlesearch:", err)
			os.Exit(1)
		}
	}

	if *screenshot != "" {
		output, err := runPlugin(c, "screenshot", map[string]any{
			"urls": []string{*screenshot},
		})
		if err != nil {
			fmt.Println("failed to run screenshot:", err)
			os.Exit(1)
		}
		if err := downloadScreenshots(c, output); err != nil {
			fmt.Println("failed to download screenshots:", err)
			os.Exit(1)
		}
	}

	if *test {
		runTest(c)
	}
}

// runPlugin runs a single plugin job and prints its output as indented JSON.
func runPlugin(c *client.Client, pluginName string, params map[string]any) (map[string]any, error) {
	output, err := c.RunPlugin(pluginName, params)
	if err != nil {
		return nil, err
	}
	encoded, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to JSON encode output: %w", err)
	}
	fmt.Println(string(encoded))
	return output, nil
}

// downloadScreenshots downloads every file referenced by the screenshot plugin
// output and saves it in the current directory under its server-side name.
func downloadScreenshots(c *client.Client, output map[string]any) error {
	result, ok := output["screenshot"].(map[string]any)
	if !ok {
		return fmt.Errorf("unexpected screenshot output: %v", output)
	}
	files, ok := result["files"].([]any)
	if !ok {
		return fmt.Errorf("unexpected screenshot files: %v", result["files"])
	}

	for _, f := range files {
		// The server returns file names, which double as file IDs.
		fileID, ok := f.(string)
		if !ok {
			return fmt.Errorf("unexpected screenshot file name: %v", f)
		}
		data, err := c.DownloadFile(fileID)
		if err != nil {
			return fmt.Errorf("failed to download %s: %w", fileID, err)
		}
		// Never let a server-provided name escape the current directory.
		name := filepath.Base(fileID)
		if err := os.WriteFile(name, data, 0o644); err != nil {
			return fmt.Errorf("failed to save %s: %w", name, err)
		}
		fmt.Printf("saved %s (%d bytes)\n", name, len(data))
	}

	return nil
}

// runTest exercises the server with concurrent plugin jobs.
func runTest(c *client.Client) {
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
	var success atomic.Int64
	var wg sync.WaitGroup
	wg.Add(numOfJobs)
	for i := range numOfJobs {
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
				success.Add(1)
			}
		}()
	}

	wg.Wait()
	fmt.Printf("successfully ran %d out of %d jobs\n", success.Load(), numOfJobs)
	fmt.Printf("success rate %.2f%%\n", float64(success.Load())/float64(numOfJobs)*100)
}
