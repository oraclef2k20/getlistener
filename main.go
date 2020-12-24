package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"regexp"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling/types"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
)

func main() {

	stage := flag.String("stage", "staging", "stage")

	flag.Parse()
	args := flag.Args()
	project := args[0]

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	elbv2 := elasticloadbalancingv2.NewFromConfig(cfg)

	auto := autoscaling.NewFromConfig(cfg)

	//fmt.Println(project, *stage)
	asgs := describeTags(auto, project, *stage)
	//fmt.Println("asgs", asgs)

	tgsmap := describeAutoScalingGroup(auto, asgs)
	for asgname, tgs := range tgsmap {
		lbs := describeTargetGroups(elbv2, tgs)
		fmt.Println("asgname: ", asgname)
		for tg, lb := range lbs {

			//fmt.Println(v2)
			lsts := describeListerners(elbv2, lb)

			for _, lst := range lsts {
				lsts2 := describeRules(elbv2, lst, tg)
				if len(lsts2) != 0 {

					fmt.Println(lsts2[0])
				}
			}
		}
	}
}

func describeTags(svc *autoscaling.Client, project string, stage string) []string {

	param := &autoscaling.DescribeTagsInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("value"),
				Values: []string{project},
			},
		},
	}

	p := autoscaling.NewDescribeTagsPaginator(svc, param)

	r := regexp.MustCompile(stage)
	var i int
	asgs := []string{}
	for p.HasMorePages() {
		i++

		page, err := p.NextPage(context.TODO())
		if err != nil {
			fmt.Println(err)
		}

		for _, v := range page.Tags {
			if r.MatchString(*v.ResourceId) {
				//fmt.Println(aws.ToString(v.ResourceId))
				asgs = append(asgs, aws.ToString(v.ResourceId))
			}
		}
	}
	return uniq(asgs)
}

func describeAutoScalingGroup(svc *autoscaling.Client, asgs []string) map[string][]string {
	param := &autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: asgs,
	}

	p := autoscaling.NewDescribeAutoScalingGroupsPaginator(svc, param)

	//	tgs:=[]map[string]string{}
	tgs := map[string][]string{}
	var i int
	for p.HasMorePages() {
		i++

		page, err := p.NextPage(context.TODO())
		if err != nil {
			fmt.Println(err)
		}

		for _, v := range page.AutoScalingGroups {

			//fmt.Println(v.TargetGroupARNs)
			for _, tg := range v.TargetGroupARNs {
				//		fmt.Println(tg)
				tgs[aws.ToString(v.AutoScalingGroupName)] = append(tgs[aws.ToString(v.AutoScalingGroupName)], tg)
			}
		}
	}
	return tgs

}

func describeListerners(svc *elasticloadbalancingv2.Client, lb string) []string {
	param := &elasticloadbalancingv2.DescribeListenersInput{
		LoadBalancerArn: aws.String(lb),
	}

	p := elasticloadbalancingv2.NewDescribeListenersPaginator(svc, param)

	lsts := []string{}
	var i int
	for p.HasMorePages() {
		i++
		page, err := p.NextPage(context.TODO())
		if err != nil {
			fmt.Println(err)

		}
		for _, v := range page.Listeners {
			//fmt.Println(*v.ListenerArn)
			lsts = append(lsts, *v.ListenerArn)
		}
	}
	return lsts
}

func describeRules(svc *elasticloadbalancingv2.Client, lst string, tg string) []string {
	param := &elasticloadbalancingv2.DescribeRulesInput{
		ListenerArn: aws.String(lst),
	}

	res, err := svc.DescribeRules(context.TODO(), param)
	if err != nil {
		fmt.Println(err)
	}

	lsts := []string{}
	for _, v := range res.Rules {
		for _, v2 := range v.Actions {
			//fmt.Println(v2.ForwardConfig)
			if v2.ForwardConfig != nil {

				for _, v3 := range v2.ForwardConfig.TargetGroups {
					//fmt.Println(aws.ToString(v3.TargetGroupArn))
					//fmt.Println("tg", tg)

					if aws.ToString(v3.TargetGroupArn) == tg {
						//fmt.Println(lst)
						lsts = append(lsts, lst)
					}
				}
			}
		}
	}
	return uniq(lsts)
}

func describeTargetGroups(svc *elasticloadbalancingv2.Client, tgs []string) map[string]string {
	param := &elasticloadbalancingv2.DescribeTargetGroupsInput{
		TargetGroupArns: tgs,
	}
	p := elasticloadbalancingv2.NewDescribeTargetGroupsPaginator(svc, param)

	var i int
	lbs := map[string]string{}
	for p.HasMorePages() {
		i++

		page, err := p.NextPage(context.TODO())
		if err != nil {
			fmt.Println(err)

		}
		for _, v := range page.TargetGroups {
			//	fmt.Println(v.LoadBalancerArns)
			for _, v2 := range v.LoadBalancerArns {
				lbs[*v.TargetGroupArn] = v2
			}
		}
	}

	return lbs

}

func uniq(data []string) []string {

	m := make(map[string]struct{})
	// make data to unique
	for _, v := range data {
		m[v] = struct{}{}
	}

	uniq := []string{}
	for i := range m {
		uniq = append(uniq, i)
	}
	return uniq
}
