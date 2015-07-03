(function() {
  var app = angular.module('mozWebApp', []);

  app.service('MozWebApp', ['$rootScope', function($rootScope) {
    this.checkInstalled = function(onsuccess) {
      if (navigator.mozApps === undefined) {
        console.log("[mozWebApp] mozApps not available.");
        return;
      }

      var checkInstalledReq = navigator.mozApps.checkInstalled("/manifest.webapp");
      checkInstalledReq.onsuccess = function() {
        if (typeof(onsuccess) === 'function')
          $rootScope.$apply(onsuccess(checkInstalledReq.result));
      };
    };

    this.install = function(onsuccess) {
      if (navigator.mozApps === undefined) {
        console.log("[mozWebApp] mozApps not available.");
        return;
      }

      var installReq = navigator.mozApps.install(location.protocol + location.host + "/manifest.webapp");
      installReq.onsuccess = function() {
        if (typeof(onsuccess) === 'function') {
          $rootScope.$apply(onsuccess());
        }
      };
    };
  }]);
})();
