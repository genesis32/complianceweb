
axios.defaults.xsrfCookieName = "X-CSRF-Token"
axios.defaults.xsrfHeaderName = "X-CSRF-Token"

const transport = axios.create({
    timeout: 10000,
    withCredentials: true
});

Vue.component('org-item', {
    props: ['id', 'name', 'children'],
    template: `
    <div>
        <div>
            {{ id }} {{ name }}
        </div>
        <org-item v-for="child in children" v-bind:key="child.id" :id="child.id" :children="child.children" :name="child.name"></org-item>
    </div>
    `
})

var orgListView = new Vue({
    el: '#org-item-view',
    delimiters: ['${', '}'],
    mounted: function () {
        var that = this;
        transport.get('/api/organizations', {
        })
            .then(function (response) {
                console.log(response);
                that.tree = response.data;
            })
            .catch(function (error) {
                console.log(error);
            });
    },
    data: {
        tree: { name: "abc", children: [] }
    }
})
