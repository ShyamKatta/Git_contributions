package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

type contributions struct {
	totalContributions int
	details            struct {
		created      int
		commits      int
		pullRequests int
		issues       int
	}
}

/* Retrives organization list of user */
func fetchOrgsRepos(done chan<- bool, orgMapChan chan<- map[string]bool, client *github.Client) {
	orgs, _, err1 := client.Organizations.List(context.Background(), "shyamkatta", nil)
	if err1 == nil {
		fmt.Println(orgs, " is list of orgs")
	} else {
		fmt.Println(err1, " is err1 ")
	}
	done <- true
	// need to populate the orgMapChan which will be used to populate repos
}

func collectUserRepoList(done chan<- bool, repoSetChan chan<- map[string]bool, resultMapChan chan<- map[string]contributions, client *github.Client) {
	// Handle Pagination
	repoSet := make(map[string]bool)
	resultMap := make(map[string]contributions)
	var allRepos []*github.Repository     // array of pointers to github.Repository structure
	opt := &github.RepositoryListOptions{ // reference [2]
		ListOptions: github.ListOptions{PerPage: 30},
	}
	for {
		repos, resp, err2 := client.Repositories.List(context.Background(), "", opt)
		if err2 != nil {
			return
		}
		allRepos = append(allRepos, repos...)
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	for _, item := range allRepos {
		if _, err := repoSet[*item.FullName]; !err {
			if !*item.Fork { // add a repo only if it is not a forked one
				//fmt.Println("adding repo ", *item)
				repoSet[*item.FullName] = true
			} else {
				// add the souce repository to repository set
				//repoSet[*item.Source.FullName] = true
				//fmt.Println(*item)
				// check if this repo is owned by current user
				repoName := *item.FullName
				if strings.EqualFold(strings.Split(repoName, "/")[0], "shyamkatta") {
					// add this repo created time to final result map, since creation of repos also count as contribution
					contribTime := *item.CreatedAt
					contrib, _ := resultMap[contribTime.Format("2006-01-02")]
					contrib.totalContributions++
					contrib.details.created++
					resultMap[contribTime.Format("2006-01-02")] = contrib
				}
			}
		}
	}
	fmt.Println("user repo list collection complete len -> ", len(repoSet), len(resultMap))
	//fmt.Println(repoSet)
	done <- true
	repoSetChan <- repoSet
	resultMapChan <- resultMap
	return
}

func collectPersonalRepos(done chan<- bool, repoSetChan chan<- map[string]bool) {
	// Need to handle pagination if the user has more than 100 repos or per_page supports only 30 repos
	repoSet := make(map[string]bool)
	res, err := http.Get("https://api.github.com/users/shyamkatta/repos?per_page=100")
	if err != nil {
		fmt.Print(err.Error())
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Print(err.Error())
	}

	var t []interface{}
	newErr := json.Unmarshal([]byte(body), &t)
	fmt.Println(newErr)
	// Add all repos even if the repo is forked one, bcaz we have a contribution for created times of user repos
	for _, m := range t {
		a := m.(map[string]interface{})
		if _, err := repoSet[a["full_name"].(string)]; !err {
			repoSet[a["full_name"].(string)] = true
		}
	}
	fmt.Println("user personal repos - ", repoSet)
	repoSetChan <- repoSet
	done <- true
}

func populateCommitsOfSavedRepos(done chan<- bool, userName string, repoSet map[string]bool, resultMap map[string]contributions, resultMapChan chan<- map[string]contributions, client *github.Client) {
	//fmt.Print(len(repoSet))
	opt1 := &github.CommitsListOptions{ // reference [2]
		ListOptions: github.ListOptions{PerPage: 100},
		Author:      userName,
	}
	for k := range repoSet {
		splitString := strings.Split(k, "/")
		//fmt.Println(splitString[0], "--__--", splitString[1])
		//for {
		repos, _, err2 := client.Repositories.ListCommits(context.Background(), splitString[0], splitString[1], opt1)
		if err2 != nil {
			fmt.Print("Error in listing commits of a repo", k)
			return
		}
		var contribTime time.Time
		for _, item := range repos {
			commiter := *item.GetCommit().GetCommitter()
			contribTime = commiter.GetDate()

			contrib, _ := resultMap[contribTime.Format("2006-01-02")]
			contrib.totalContributions++
			contrib.details.commits++
			resultMap[contribTime.Format("2006-01-02")] = contrib
		}
	}
	// if resp.NextPage == 0 {
	// 	break
	// }
	// opt.Page = resp.NextPage
	// break
	//}
	fmt.Println(len(resultMap), " --> map is of commit length")
	done <- true
	resultMapChan <- resultMap
	return
}

func populatePullsOfSavedRepos(done chan<- bool, repoSet map[string]bool, resultMap map[string]contributions, resultMapChan chan<- map[string]contributions, client *github.Client) {
	// handle pagination here
	opt := &github.PullRequestListOptions{ // reference [2]
		ListOptions: github.ListOptions{PerPage: 100},
		State:       "closed",
	}
	for k := range repoSet {
		splitString := strings.Split(k, "/")

		for {
			pulls, resp, err := client.PullRequests.List(context.Background(), splitString[0], splitString[1], opt)
			if err != nil {
				fmt.Println("_+_()--------**********erroro ")
				return
			}
			var contribTime time.Time
			for _, item := range pulls {
				//fmt.Println(*item.User.Login)
				if pullAuthor := *item.User.Login; strings.EqualFold(pullAuthor, "shyamkatta") {
					contribTime = *item.CreatedAt
					contrib, _ := resultMap[contribTime.Format("2006-01-02")]
					contrib.details.pullRequests++
					contrib.totalContributions++
					resultMap[contribTime.Format("2006-01-02")] = contrib
				}
			}
			if resp.NextPage == 0 {
				break
			}
			opt.Page = resp.NextPage
		}
	}
	fmt.Println(len(resultMap), " --> map is of commit length")
	done <- true
	resultMapChan <- resultMap
	return
}

func populateIssuesOfSavedRepos(done chan<- bool, resultMap map[string]contributions, resultMapChan chan<- map[string]contributions, client *github.Client) {
	// handle pagination here
	opt := &github.IssueListOptions{ // reference [2]
		ListOptions: github.ListOptions{PerPage: 30},
	}
	for {
		repos, resp, err := client.Issues.List(context.Background(), true, opt)
		if err != nil {
			return
		}
		var contribTime time.Time
		fmt.Println(len(repos), " total issues received")
		for _, item := range repos {
			contribTime = *item.CreatedAt
			contrib, _ := resultMap[contribTime.Format("2006-01-02")]
			contrib.details.issues++
			contrib.totalContributions++
			resultMap[contribTime.Format("2006-01-02")] = contrib
		}
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	done <- true
	resultMapChan <- resultMap
	return
}

func main() {

	// handle the error if argument in short and exit

	/* Add all the repos to a map implementation of a set - repoSet */
	repoSet := make(map[string]bool)
	resultMap := make(map[string]contributions)
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: "your access code"},
	)
	tc := oauth2.NewClient(context.Background(), ts)
	client := github.NewClient(tc)

	//creating channels
	done := make(chan bool, 5)
	orgMapChan := make(chan map[string]bool, 1)
	resultMapChan := make(chan map[string]contributions, 3)
	repoSetChan := make(chan map[string]bool, 4)

	fmt.Println(reflect.TypeOf(client))

	go fetchOrgsRepos(done, orgMapChan, client)
	close(orgMapChan)
	/* Collect user repos list */
	<-done
	go collectUserRepoList(done, repoSetChan, resultMapChan, client)
	//result set, repo map
	<-done
	// localRepoSet := <-repoSetChan
	// localResultMap := <-resultMapChan
	// fmt.Println(len(localRepoSet), " ignore")
	//		uncomment here
	localRepoSet := <-repoSetChan
	for key, value := range localRepoSet {
		repoSet[key] = value
	}
	localResultMap := <-resultMapChan
	for key, value := range localResultMap {
		resultMap[key] = value
	}
	// fmt.Println("after adding globally from function len is -->", len(repoSet), len(resultMap))
	/* End of collecting user repos list */

	/* Get commits from all the saved repos */
	// need to include pagination if user has more than 100 commtis in a repo

	/*go collectPersonalRepos(done, repoSetChan)
	<-done
	personalRepoSet := <-repoSetChan */
	/* End of collencting this repos */
	//<-done

	/* Below modules can be executed concurrently, by creating 3 channels and combining the info from each channel to a existing resultMap created for create repos*/
	go populateCommitsOfSavedRepos(done, "shyamkatta", repoSet, resultMap, resultMapChan, client)
	<-done
	localResultMap = <-resultMapChan
	//fmt.Print(len(localResultMap), " ----****")
	for key, value := range localResultMap {
		resultMap[key] = value
	}
	/* End of retriving commits */
	go populateIssuesOfSavedRepos(done, resultMap, resultMapChan, client)
	<-done
	localResultMap = <-resultMapChan
	for key, value := range localResultMap {
		resultMap[key] = value
	}
	go populatePullsOfSavedRepos(done, repoSet, resultMap, resultMapChan, client)
	<-done
	localResultMap = <-resultMapChan
	for key, value := range localResultMap {
		resultMap[key] = value
	}

	fmt.Println(resultMap, "--> map printed")
	/* Collect user personal repos from git api */

	close(resultMapChan)
	close(repoSetChan)
	sum := 0
	for _, v := range resultMap {
		//fmt.Print("\t", v.totalContributions, " ,sum is ", sum)
		sum += v.totalContributions
	}
	fmt.Print("total contrib - ", sum)
	fmt.Println("\n", len(repoSet))
	//fmt.Println("map set : ", repoSet)

	/* References
		Go-github -
		https://medium.com/@durgaprasadbudhwani/playing-with-github-api-with-go-github-golang-library-83e28b2ff093
		[2] https://godoc.org/github.com/google/go-github/github#hdr-Pagination

		Get a list of repos from github - regular JSON
		https://flaviocopes.com/go-github-api/

		Error - cannot convert data (type interface {}) to type string: need type assertion
		https://stackoverflow.com/questions/14289256/cannot-convert-data-type-interface-to-type-string-need-type-assertion

		Error - spec: cannot assign to a field of a map element directly:
		assignments might have mnultiple reutrn values, hence must assign to variable and use attribs of that variable
		https://github.com/golang/go/issues/3117s

		https://stackoverflow.com/questions/25318154/convert-utc-to-local-time-go

		Go-github
		https://godoc.org/github.com/google/go-github/github#Repository
		https://github.com/google/go-github/blob/master/github/github-accessors.go#L696

	    http://jsonviewer.stack.hu


	    // Commits
	    https://godoc.org/github.com/google/go-github/github#RepositoriesService.ListCommits
	    get pull req of repo
	    https://godoc.org/github.com/google/go-github/github#PullRequestsService.List
	    Repository service
		  https://godoc.org/github.com/google/go-github/github#RepositoriesService.List
	*/
}
