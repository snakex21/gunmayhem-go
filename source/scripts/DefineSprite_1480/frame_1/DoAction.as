function update()
{
   lastlife.gotoAndStop(1);
   updated = true;
   player.head.gotoAndStop(colornumber + 1);
   player.body.gotoAndStop(colornumber + 1);
   player.leg1.leg.gotoAndStop(colornumber + 1);
   player.leg2.leg.gotoAndStop(colornumber + 1);
   player.hand1.hand.gotoAndStop(colornumber + 1);
   player.hand2.hand.gotoAndStop(colornumber + 1);
   if(_root.gamemode == 4)
   {
      displayheader.gotoAndStop(2);
      flawless._alpha = 0;
   }
   else if(_root.gamemode == 5)
   {
      displayheader.gotoAndStop(3);
      flawless._alpha = 0;
   }
}
switch(_name)
{
   case "card1":
      colornumber = _root.p1color;
      inputname.text = _root.p1name;
      player.shirt.gotoAndStop(_root.p1shirt);
      player.hat.gotoAndStop(_root.p1hat);
      player.eyes.gotoAndStop(_root.p1hat);
      bg.gotoAndStop(_root.p1color + 1);
      target = _root.player1;
      break;
   case "card2":
      colornumber = _root.p2color;
      inputname.text = _root.p2name;
      player.shirt.gotoAndStop(_root.p2shirt);
      player.hat.gotoAndStop(_root.p2hat);
      player.eyes.gotoAndStop(_root.p2hat);
      bg.gotoAndStop(_root.p2color + 1);
      target = _root.player2;
      break;
   case "card3":
      colornumber = _root.p3color;
      inputname.text = _root.p3name;
      player.shirt.gotoAndStop(_root.p3shirt);
      player.hat.gotoAndStop(_root.p3hat);
      player.eyes.gotoAndStop(_root.p3hat);
      bg.gotoAndStop(_root.p3color + 1);
      target = _root.player3;
      break;
   case "card4":
      colornumber = _root.p4color;
      inputname.text = _root.p4name;
      player.shirt.gotoAndStop(_root.p4shirt);
      player.hat.gotoAndStop(_root.p4hat);
      player.eyes.gotoAndStop(_root.p4hat);
      bg.gotoAndStop(_root.p4color + 1);
      target = _root.player4;
}
updated = false;
lastlevel = target.currentlevel;
this.onEnterFrame = function()
{
   if(_root.gamemode != 4 && _root.gamemode != 5)
   {
      if(livedisplay.text == _root.totallives && flawless._currentframe != 1)
      {
         flawless.gotoAndStop(1);
      }
      else if(livedisplay.text == _root.totallives - 1 && flawless._currentframe != 2)
      {
         flawless.gotoAndStop(2);
      }
      else if(parseInt(livedisplay.text) > _root.totallives && flawless._currentframe != 4)
      {
         flawless.gotoAndStop(4);
      }
      else if(parseInt(livedisplay.text) < _root.totallives - 1 && flawless._currentframe != 3)
      {
         flawless.gotoAndStop(3);
      }
      if(livedisplay.text == "0" && player._currentframe == 1)
      {
         player.gotoAndStop(2);
         ammodisplay.text = "";
         gunname.text = "";
      }
      if(livedisplay.text == "1" && lastlife._currentframe == 1)
      {
         lastlife.play();
      }
      if(livedisplay.text == "0" && lastlife._currentframe == 90)
      {
         lastlife.play();
      }
   }
   else
   {
      if(lastlevel < target.currentlevel)
      {
         lastlife.gotoAndPlay(181);
      }
      lastlevel = target.currentlevel;
   }
   if(!updated)
   {
      update();
   }
};
