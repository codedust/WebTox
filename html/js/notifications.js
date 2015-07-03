(function() {
  var app = angular.module('notifications', []);

  app.service('Notifications', ['$rootScope', function($rootScope) {
    var notify = function(title, body, tag, callback) {
      var notification = new Notification(title, {
        body: body,
        tag: tag,
        icon: '/img/favicon.png'
      }).onclick = function() {
        if (typeof(callback) === 'function')
          $rootScope.$apply(callback());
      };
    };

    this.show = function(title, body, tag, callback) {
      if (title === undefined || title === '') {
        console.log('[Notifications] Title has to be set.');
        return;
      }

      if (!("Notification" in window)) {
        if (body === undefined)
          body = '';

        alert(title + "\n" + body);

        if (typeof(callback) === 'function')
          $rootScope.$apply(callback());

      } else if (Notification.permission === "granted") {
        notify(title, body, tag, callback);

      } else if (Notification.permission !== 'denied') {
        Notification.requestPermission(function(permission) {
          if (!('permission' in Notification)) {
            Notification.permission = permission;
          }
          if (permission === "granted") {
            notify(title, body, tag, callback);
          }
        });
      }
    };
  }]);
})();
