
axios.defaults.xsrfCookieName = "X-CSRF-Token"
axios.defaults.xsrfHeaderName = "X-CSRF-Token"

const transport = axios.create({
    timeout: 10000,
    withCredentials: true
});

Vue.component('useradd-widget', {
    data: function () {
        return {
            accountName: "John Doe"
        }
    },
    methods: {
        addUser: function (event) {
            transport.post('/webapp/userJSON', {
                name: 'Fred Flintstone'
              })
              .then(function (response) {
                console.log(response);
              })
              .catch(function (error) {
                console.log(error);
              });
            alert('Add ' + this.accountName);
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
