## Clanker

Use [crush](https://charm.land/crush/) with the `GLM-5.2` model ([hyper](https://hyper.charm.land)).
Make sure you have the `carapace` and `jq` skills.

1. tell it what is wrong and what you expect
2. verify with `cmd/carapace-jq` and see if the change broke sth. else
3. if so, use the same session to tell it what broke (this can take 2-3 iterations)
4. tell it to add tests
