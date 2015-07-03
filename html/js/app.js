/*
  WebTox - A web based graphical user interface for Tox
  Copyright (C) 2014 WebTox authors and contributers

  This file is part of WebTox.

  WebTox is free software: you can redistribute it and/or modify
  it under the terms of the GNU General Public License as published by
  the Free Software Foundation, either version 3 of the License, or
  (at your option) any later version.

  WebTox is distributed in the hope that it will be useful,
  but WITHOUT ANY WARRANTY; without even the implied warranty of
  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
  GNU General Public License for more details.

  You should have received a copy of the GNU General Public License
  along with WebTox.  If not, see <http://www.gnu.org/licenses/>.
*/

(function() {
  var app = angular.module('webtox', ['fullscreen', 'mozWebApp', 'notifications', 'websocket']);

  app.controller('webtoxCtrl', ['$scope', '$http', 'Fullscreen', 'MozWebApp', 'Notifications', 'WS', function($scope, $http, FullscreenService, WebApp, Notifications, WS) {
    'use strict';

    $scope.goFullscreen = FullscreenService.goFullscreen;

    $scope.notImplemented = function() {
      alert("This feature is not implemented yet. :( Sorry.");
    };

    // == initialise scope ==
    $scope.active_mainview = 'welcome';
    $scope.profile = {
      username: "Loading...",
      status_msg: "Loading...",
      tox_id: "Loading...",
    };
    $scope.contacts = [];
    $scope.activecontactindex = -1;
    $scope.messagetosend = '';
    $scope.new_friend_request = {
      friend_id: '',
      message: '',
    };
    $scope.settings = {};
    $scope.curDate = Date.now(); // current unix timestap used to work around caching

    var getContactIndexByNum = function(num) {
      for (var i in $scope.contacts)
        if ($scope.contacts[i].number === num) return i;
      return -1;
    };

    $scope.setUsername = function(username) {
      $http.post('api/post/username', {
        username: username
      }).success(function() {
        fetchProfile();
      }).error(function() {
        fetchProfile();
      });
    };

    $scope.setStatusMsg = function(status_msg) {
      $http.post('api/post/statusmessage', {
        status_msg: status_msg
      }).success(function() {
        fetchProfile();
      }).error(function() {
        fetchProfile();
      });
    };

    $scope.setUserStatus = function(status) {
      $http.post('api/post/status', {
        status: status
      }).success(function() {
        fetchProfile();
      }).error(function() {
        fetchProfile();
      });
    };

    $scope.showChat = function(friendnumber) {
      var i = getContactIndexByNum(friendnumber);
      if (i != -1) {
        $scope.activecontactindex = i;
        $scope.active_mainview = 'chat';
        sendMessageRead(friendnumber);

        window.setTimeout(function() {
          $("#mainview-chat-body").scrollTop($("#mainview-chat-body").prop("scrollHeight"));
        }, 10);
      }
    };

    // == Settings ==
    $scope.showSettings = function() {
      $scope.active_mainview = 'settings';
    };

    // == Messages ==
    $scope.sendMessage = function() {
      if ($scope.messagetosend.length === 0)
        return;

      if (!$scope.contacts[$scope.activecontactindex].online) {
        // TODO cache messages server-side until the user gets online again...
        alert("User is offline. :(");
        return;
      }

      $http.post('api/post/message', {
        friend: $scope.contacts[$scope.activecontactindex].number,
        message: $scope.messagetosend
      }).error(function() {
        // TODO
      });

      $scope.contacts[$scope.activecontactindex].chat.unshift({
        "isIncoming": false,
        "isAction": false,
        "message": $scope.messagetosend.replace(/\n/g, "<br>"),
        "time": Date.now()
      });
      $scope.contacts[$scope.activecontactindex].last_msg_read = Date.now();
      $scope.messagetosend = '';

      $("#mainview-chat-body").animate({
        "scrollTop": $("#mainview-chat-body").prop("scrollHeight")
      }, 1000);
    };

    var sendMessageRead = function(friendnumber) {
      $http.post('api/post/message_read_receipt', {
        friend: friendnumber
      }).success(function() {
        $scope.contacts[$scope.activecontactindex].last_msg_read = Date.now();
      });
    };


    // == Friends ==
    $scope.sendFriendRequest = function(friend_id, message) {
      $http.post('api/post/friend_request', {
        friend_id: friend_id,
        message: message
      }).success(function() {
        $('#modal-friend-requests').modal('hide');
        $http.get('api/get/contactlist').success(function(data) {
          $scope.contacts = data;
        });
      }).error(function(err) {
        // TODO
        alert(err.message);
      });
    };

    $scope.deleteFriend = function(friend) {
      $http.post('api/post/delete_friend', {
        friend: friend
      }).success(function() {
        $('#modal-friend-del').modal('hide');
        $http.get('api/get/contactlist').success(function(data) {
          $scope.contacts = data;
        });
      }).error(function(err) {
        // TODO
        alert(err.message);
      });
    };

    // == Event handlers ==
    $('#profile-card-back-button').click(function() {
      $('#profile-card, #contact-list-wrapper, #button-panel').removeClass('translate75left');
      $('#mainview').removeClass('translate100left');
      $('#profile-card-back-button').hide();
    });

    $('#contact-list-wrapper').click(function() {
      if ($(window).width() < 768) {
        $('#profile-card, #contact-list-wrapper, #button-panel').addClass('translate75left');
        $('#mainview').addClass('translate100left');
        $('#profile-card-back-button').show();
      }
    });

    $("#mainview-chat-footer-textarea-wrapper textarea").keyup(function(event) {
      if (event.which == 13 && event.shiftKey !== true) {
        $scope.sendMessage();
      }
    });

    $('#inputAuthUser').change(function() {
      $(this).parent().next().find('button').show();
    }).keyup(function() {
      $(this).parent().next().find('button').show();
    }).parent().next().find('button').click(function() {
      $http.post('api/post/settings_auth_user', {
        username: $('#inputAuthUser').val()
      }).success(function() {
        $('#inputAuthUser').parent().next().find('button').hide();
      });
    });

    $('#inputAuthPass').change(function() {
      $(this).parent().next().find('button').show();
    }).keyup(function() {
      $(this).parent().next().find('button').show();
    }).parent().next().find('button').click(function() {
      $http.post('api/post/settings_auth_pass', {
        password: $('#inputAuthPass').val()
      }).success(function() {
        $('#inputAuthPass').parent().next().find('button').hide();
        $('#inputAuthPass').val('');
      });
    });

    $('#checkbox-notifications').change(function() {
      $http.post('api/post/keyValue', {
        key: 'settings_notifications_enabled',
        value: $('#checkbox-notifications').prop('checked').toString()
      }).error(function() {
        fetchSettings();
      });
    });

    $('#checkbox-away-on-disconnect').change(function() {
      $http.post('api/post/keyValue', {
        key: 'settings_away_on_disconnect',
        value: $('#checkbox-away-on-disconnect').prop('checked').toString()
      }).error(function() {
        fetchSettings();
      });
    });

    // == WebApp Installation ==
    $scope.appInstallationStatus = 'unknown';

    WebApp.checkInstalled(function(installed) {
      $scope.appInstallationStatus = (installed) ? 'installed' : 'notinstalled';
    });

    $scope.installWebApp = function() {
      WebApp.install(function() {
        $scope.appInstallationStatus = 'installed';
      });
    };

    // == fetch data from the server ==
    var fetchSettings = function() {
      $http.get('api/get/settings').success(function(data) {
        $scope.settings = data;
      });
    };

    var fetchProfile = function() {
      $http.get('api/get/profile').success(function(data) {
        $scope.profile = data;
      });
    };

    var fetchContactlist = function() {
      $http.get('api/get/contactlist').success(function(data) {
        $scope.contacts = data;
      });
    };

    // == WebSocket connection ==
    WS.registerHandler('friend_message', function(data) {
      var i = getContactIndexByNum(data.friend);
      if (i >= 0 && i < $scope.contacts.length) {
        $scope.contacts[i].chat.unshift({
          "message": data.message,
          "isIncoming": true,
          "isAction": data.isAction,
          "time": data.time
        });
        if ($scope.settings.notifications_enabled) {
          Notifications.show($scope.contacts[i].name, data.message, "friend_message"+$scope.contacts[i].number, function() {
            $scope.showChat(data.friend);
          });
        }

        $("#mainview-chat-body").animate({
          "scrollTop": $("#mainview-chat-body").prop("scrollHeight")
        }, 1000);
      }
    });

    WS.registerHandler('name_changed', function(data) {
      var i = getContactIndexByNum(data.friend);
      if (i >= 0 && i < $scope.contacts.length)
        $scope.contacts[i].name = data.name;
    });

    WS.registerHandler('status_message_changed', function(data) {
      var i = getContactIndexByNum(data.friend);
      if (i >= 0 && i < $scope.contacts.length)
        $scope.contacts[i].status_msg = data.status_msg;
    });

    WS.registerHandler('status_changed', function(data) {
      var i = getContactIndexByNum(data.friend);
      if (i >= 0 && i < $scope.contacts.length)
        $scope.contacts[i].status = data.status;
    });

    WS.registerHandler('connection_status', function(data) {
      var i = getContactIndexByNum(data.friend);
      $scope.contacts[i].online = data.online;
      if ($scope.settings.notifications_enabled) {
        Notifications.show($scope.contacts[i].name, "is now " + (data.online ? 'online' : 'offline'), "connection_status"+$scope.contacts[i].number);
      }
    });

    WS.registerHandler('profile_update', fetchProfile);
    WS.registerHandler('friendlist_update', fetchContactlist);

    WS.registerHandler('avatar_update', function() {
      $scope.curDate = Date.now(); // reload avatar images
    });

    var onopen = function(event) {
      console.log("WebSocket connection established.");
      $('#modal-connection-error').modal('hide');
      fetchProfile();
      fetchContactlist();
      fetchSettings();
      $scope.$apply();
    };

    var onclose = function() {
      console.log("WebSocket connection closed!");
      $('.modal.info, .modal.warning').modal('hide');
      $('#modal-connection-error').modal('show');
      window.setTimeout($scope.ws_create, 5000);
    };

    WS.newConnection(onopen, onclose);

  }]);
})();
