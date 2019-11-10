function loadUserList() {
    alert('hello world');
}

var example1 = new Vue({
    el: '#example-1',
    delimiters: ['${', '}'],
    data: {
      items: [
        { message: 'Foo' },
        { message: 'Bar' }
      ]
    }
  })