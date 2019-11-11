
axios.defaults.xsrfCookieName = "X-CSRF-Token"
axios.defaults.xsrfHeaderName = "X-CSRF-Token"

const transport = axios.create({
    timeout: 10000,
    withCredentials: true
});

Vue.component('useradd-widget', {
    props: ['orgid'],
    data: function () {
        return {
            accountName: "John Doe"
        }
    },
    methods: {
        addUser: function (event) {
            var that=this;
            transport.post('/webapp/userJSON', {
                organizationId: this.orgid,
                name: this.accountName
              })
              .then(function (response) {
                alert('Added ' + that.accountName);
                console.log(response);
              })
              .catch(function (error) {
                console.log(error);
              });
        }
    },
    template: `
    <div>
        <input type="text" name="accountName" v-model="accountName"></input>
        <button name="addUser" v-on:click="addUser">Add User</button>
    </div>
    `
})

Vue.component('userlist-widget', {
    data: function () {
        return {
            items: [
                { message: 'Foo' },
                { message: 'Bar' }
            ]
        }
    },
    methods: {

    },
    template: `
    <div>
        <ul id="example-1">
            <li v-for="item in items">
                {{ item.message }}
            </li>
        </ul>
    </div>
    `
})

var userManagement = new Vue({
    el: '#user-management',
    delimiters: ['${', '}'],
    data: {
    },
    methods: {
    }
})
